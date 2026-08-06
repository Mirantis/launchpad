#!/usr/bin/env bash
#
# Delete stale smoke-test VPCs and everything inside them.
#
# A VPC is deleted only when all of the following hold:
#   * it is not the default VPC
#   * it belongs to this repo's smoke tests -- see the ownership gate below
#   * it does not carry the persist tag (see PERSIST_TAG below)
#   * it has a "created" tag (written by the terraform modules) whose timestamp
#     is older than MAX_AGE_HOURS
#
# VPCs without a "created" tag are never deleted -- their age cannot be
# established from the EC2 API, and guessing is not worth the blast radius.
# They are reported instead so a human can look.
#
# Why this exists: when a smoke test times out, `go test` panics and kills the
# process, so terratest's `defer terraform.Destroy` never runs and the whole
# stack leaks. Enough leaks and the account hits VpcLimitExceeded, which then
# fails every subsequent smoke run for reasons unrelated to the code under test.
#
# Usage:
#   DRY_RUN=true ./cleanup-stale-vpcs.sh     # report only, delete nothing
#   MAX_AGE_HOURS=48 ./cleanup-stale-vpcs.sh
#
set -euo pipefail

MAX_AGE_HOURS="${MAX_AGE_HOURS:-24}"
DRY_RUN="${DRY_RUN:-false}"
# Presence of this tag key protects a VPC regardless of its value, so
# `launchpad-persist=investigating PRODENG-1234` works as well as `=true`.
# Fail-safe by design: a typo in the value must not cause a deletion.
PERSIST_TAG="${PERSIST_TAG:-launchpad-persist}"
# Ownership gate: a VPC is only ever a deletion candidate if it carries this
# tag key, or if its `stack` tag starts with this prefix. Both signals are
# needed because neither is reliable on its own -- see the selector below.
SMOKE_TAG="${SMOKE_TAG:-launchpad-smoke-test}"
STACK_PREFIX="${STACK_PREFIX:-smoke-}"

# workflow_dispatch inputs arrive as free-form strings, so validate before the
# selector float()s this and dies with a traceback. Zero is rejected too: it
# would sweep up stacks belonging to smoke runs that are still in flight.
MAX_AGE_HOURS="$(python3 -c "
import sys
try:
    v = float(sys.argv[1])
except ValueError:
    sys.exit(f'ERROR: MAX_AGE_HOURS must be a number, got {sys.argv[1]!r}')
if v <= 0:
    sys.exit(f'ERROR: MAX_AGE_HOURS must be greater than zero, got {v}')
print(v)
" "$MAX_AGE_HOURS")" || exit 2

log()  { echo "[cleanup] $*"; }
run()  { if [ "$DRY_RUN" = "true" ]; then echo "[dry-run] $*"; else "$@" >/dev/null 2>&1 || true; fi; }

log "region=${AWS_REGION:-unset} max_age_hours=${MAX_AGE_HOURS} dry_run=${DRY_RUN}"
log "scope: stack prefix '${STACK_PREFIX}' or tag '${SMOKE_TAG}'; persist tag '${PERSIST_TAG}'"

vpcs_json="$(aws ec2 describe-vpcs \
  --query 'Vpcs[?IsDefault==`false`].{VpcId:VpcId,Tags:Tags}' --output json)"

mapfile -t doomed < <(python3 - "$vpcs_json" "$MAX_AGE_HOURS" "$PERSIST_TAG" "$SMOKE_TAG" "$STACK_PREFIX" <<'PY'
import json, sys, datetime
vpcs = json.loads(sys.argv[1])
max_age = float(sys.argv[2])
persist_tag, smoke_tag, stack_prefix = sys.argv[3], sys.argv[4], sys.argv[5]
now = datetime.datetime.now(datetime.timezone.utc)
for v in vpcs:
    tags = {t["Key"]: t["Value"] for t in (v.get("Tags") or [])}
    vpc, stack = v["VpcId"], tags.get("stack", "<untagged>")
    if persist_tag in tags:
        print(f"KEEP\t{vpc}\t{stack}\tpersist tag present", file=sys.stderr)
        continue
    # Ownership gate. Two independent signals, because neither is sufficient
    # alone: of the 16 leaked stacks cleaned up by hand on 2026-08-06, all 16
    # had a `stack` tag starting with "smoke-" but only 15 carried the
    # `launchpad-smoke-test` tag -- "smoke-mPOzo" had no smoke tag at all and
    # would have leaked indefinitely under a tag-only rule. Anything failing
    # both checks is not ours and is never a deletion candidate.
    if smoke_tag not in tags and not stack.startswith(stack_prefix):
        print(f"SKIP\t{vpc}\t{stack}\tnot a smoke-test stack - out of scope", file=sys.stderr)
        continue
    created = tags.get("created")
    if not created:
        print(f"SKIP\t{vpc}\t{stack}\tno 'created' tag - age unknown, not deleting", file=sys.stderr)
        continue
    try:
        age = (now - datetime.datetime.fromisoformat(created.replace("Z", "+00:00"))).total_seconds() / 3600
    except ValueError:
        print(f"SKIP\t{vpc}\t{stack}\tunparseable 'created' tag {created!r}", file=sys.stderr)
        continue
    if age < max_age:
        print(f"KEEP\t{vpc}\t{stack}\t{age:.1f}h old", file=sys.stderr)
        continue
    print(f"DELETE\t{vpc}\t{stack}\t{age:.1f}h old", file=sys.stderr)
    print(vpc)
PY
)

if [ "${#doomed[@]}" -eq 0 ]; then
  log "nothing to delete"
  exit 0
fi

log "deleting ${#doomed[@]} VPC(s)"

for vpc in "${doomed[@]}"; do
  log "=== $vpc ==="

  # 1. Auto scaling groups first: deleting instances while an ASG is live just
  #    makes it launch replacements. Match on the ASG's subnets, not on a name
  #    prefix, so renamed or oddly-named groups are still caught.
  subnets="$(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$vpc" \
    --query 'Subnets[].SubnetId' --output text 2>/dev/null || true)"
  if [ -n "$subnets" ]; then
    for asg in $(aws autoscaling describe-auto-scaling-groups \
        --query 'AutoScalingGroups[].[AutoScalingGroupName,VPCZoneIdentifier]' --output text 2>/dev/null \
        | awk -v s="$(echo "$subnets" | tr '\t' '|')" '$2 ~ s {print $1}'); do
      log "  asg  $asg"
      run aws autoscaling delete-auto-scaling-group --auto-scaling-group-name "$asg" --force-delete
    done
  fi

  # 2. Load balancers, then their target groups: both hold ENIs that block
  #    subnet deletion later on.
  for arn in $(aws elbv2 describe-load-balancers \
      --query "LoadBalancers[?VpcId=='$vpc'].LoadBalancerArn" --output text 2>/dev/null); do
    log "  lb   $arn"
    run aws elbv2 delete-load-balancer --load-balancer-arn "$arn"
  done
  for arn in $(aws elbv2 describe-target-groups \
      --query "TargetGroups[?VpcId=='$vpc'].TargetGroupArn" --output text 2>/dev/null); do
    log "  tg   $arn"
    run aws elbv2 delete-target-group --target-group-arn "$arn"
  done

  # 3. Any instances still present, including ones the ASG deletion above is
  #    already tearing down. "shutting-down" must be in this filter: the
  #    --force-delete above terminates asynchronously, so by the time we get
  #    here the ASG's instances are usually in that state rather than running.
  #    Omitting it matched nothing, skipped the wait, and left their ENIs
  #    attached, which then blocked subnet and VPC deletion.
  instances="$(aws ec2 describe-instances --filters "Name=vpc-id,Values=$vpc" \
    "Name=instance-state-name,Values=running,pending,stopping,stopped,shutting-down" \
    --query 'Reservations[].Instances[].InstanceId' --output text 2>/dev/null || true)"
  if [ -n "$instances" ]; then
    log "  terminating instances: $instances"
    run aws ec2 terminate-instances --instance-ids $instances
    if [ "$DRY_RUN" != "true" ]; then
      aws ec2 wait instance-terminated --instance-ids $instances 2>/dev/null || true
    fi
  fi

  # 4. NAT gateways and VPC endpoints also pin ENIs.
  for nat in $(aws ec2 describe-nat-gateways --filter "Name=vpc-id,Values=$vpc" \
      --query 'NatGateways[?State!=`deleted`].NatGatewayId' --output text 2>/dev/null); do
    log "  nat  $nat"
    run aws ec2 delete-nat-gateway --nat-gateway-id "$nat"
  done
  for ep in $(aws ec2 describe-vpc-endpoints --filters "Name=vpc-id,Values=$vpc" \
      --query 'VpcEndpoints[].VpcEndpointId' --output text 2>/dev/null); do
    log "  vpce $ep"
    run aws ec2 delete-vpc-endpoints --vpc-endpoint-ids "$ep"
  done

  # 5. Detached ENIs that nothing cleaned up.
  for eni in $(aws ec2 describe-network-interfaces --filters "Name=vpc-id,Values=$vpc" \
      --query 'NetworkInterfaces[?Status==`available`].NetworkInterfaceId' --output text 2>/dev/null); do
    log "  eni  $eni"
    run aws ec2 delete-network-interface --network-interface-id "$eni"
  done

  # 5b. Wait for the VPC's ENIs to drain. Terminating an instance or deleting a
  #     load balancer releases its ENIs asynchronously, and a single lingering
  #     one fails subnet and VPC deletion with a DependencyViolation. Bounded so
  #     a stuck ENI is reported rather than hanging the job.
  if [ "$DRY_RUN" != "true" ]; then
    for _ in $(seq 1 30); do
      enis_left="$(aws ec2 describe-network-interfaces --filters "Name=vpc-id,Values=$vpc" \
        --query 'length(NetworkInterfaces)' --output text 2>/dev/null || echo 0)"
      [ "$enis_left" = "0" ] && break
      log "  waiting for ${enis_left} ENI(s) to release"
      sleep 10
    done
  fi

  # 6. Networking, then the VPC itself.
  for igw in $(aws ec2 describe-internet-gateways --filters "Name=attachment.vpc-id,Values=$vpc" \
      --query 'InternetGateways[].InternetGatewayId' --output text 2>/dev/null); do
    log "  igw  $igw"
    run aws ec2 detach-internet-gateway --internet-gateway-id "$igw" --vpc-id "$vpc"
    run aws ec2 delete-internet-gateway --internet-gateway-id "$igw"
  done
  for sn in $(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$vpc" \
      --query 'Subnets[].SubnetId' --output text 2>/dev/null); do
    log "  sn   $sn"
    run aws ec2 delete-subnet --subnet-id "$sn"
  done
  for rt in $(aws ec2 describe-route-tables --filters "Name=vpc-id,Values=$vpc" \
      --query 'RouteTables[?length(Associations[?Main==`true`])==`0`].RouteTableId' --output text 2>/dev/null); do
    log "  rt   $rt"
    run aws ec2 delete-route-table --route-table-id "$rt"
  done
  for sg in $(aws ec2 describe-security-groups --filters "Name=vpc-id,Values=$vpc" \
      --query 'SecurityGroups[?GroupName!=`default`].GroupId' --output text 2>/dev/null); do
    log "  sg   $sg"
    run aws ec2 delete-security-group --group-id "$sg"
  done

  log "  vpc  $vpc"
  if [ "$DRY_RUN" = "true" ]; then
    echo "[dry-run] aws ec2 delete-vpc --vpc-id $vpc"
  elif ! aws ec2 delete-vpc --vpc-id "$vpc" >/dev/null 2>&1; then
    log "  WARNING: could not delete $vpc -- it still has dependencies, leaving it for manual review"
  fi
done

log "done. remaining VPCs: $(aws ec2 describe-vpcs --query 'length(Vpcs)' --output text)"
