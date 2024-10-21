HOSTS="$(yq -r ".spec.hosts[].ssh.address" ./launchpad.yaml)"

SSH_USER=rocky
SSH_FLAGS="-i examples/tf-aws/launchpad/ssh-keys/jn-PRODENG-2744-common.pem -o StrictHostKeyChecking=no"

# --- helpers ---

ssh() {
    local host=$1
    shift;
    local run=$@

    echo "ssh $SSH_FLAGS $SSH_USER@$host -- $run"
    #ssh $SSH_FLAGS $USER@$host -- "$run"
}

scp() {
    local host=$1
    shift;
    local file=$@

    echo "scp $SSH_FLAGS $file $SSH_USER@$host:~/$file"
    #scp $SSH_FLAGS $USER@$host $file $file
}

# --- handlers ___

sudo_prepareuser() {
    host=$1

    ssh $host "sudo useradd launchpad"
    ssh $host "sudo cp -R /home/rocky/.ssh /home/launchpad/"
    ssh $host "sudo chown -R launchpad:launchpad /home/launchpad"
}

sudo_sudowhitelist() {
    host=$1

    scp $host 50-launchpad
    ssh $host "sudo chown root:root ./50-launchpad"
    ssh $host "sudo mv ./50-launchpad /etc/sudoers.d/"
}

# --- fix all hosts ---

set +x

for host in $HOSTS
do
    #echo "#-- HOST: $host"
    ssh $host whoami

    sudo_prepareuser $host
    sudo_sudowhitelist $host
done
