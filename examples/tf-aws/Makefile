
build-infra:
	terraform init
	terraform apply -auto-approve
	terraform output -json | yq r --prettyPrint - ucp_cluster.value > cluster.yaml
	cat cluster.yaml
teardown-infra:
	terraform destroy
	rm cluster.yaml
