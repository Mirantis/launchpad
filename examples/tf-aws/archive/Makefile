
build-infra:
	terraform init
	terraform apply -auto-approve
	terraform output -json | yq r --prettyPrint - ucp_cluster.value > launchpad.yaml
	cat launchpad.yaml
teardown-infra:
	terraform destroy
	rm launchpad.yaml
