smoke-register-test: build
	./smoke_register.sh

smoke-apply-test: build
	./smoke_apply.sh

smoke-apply-upload-test: build
	./smoke_upload.sh

smoke-apply-local-repo-test: build
	./smoke_localrepo.sh

smoke-reset-local-repo-test: build
	./smoke_localrepo_reset.sh

smoke-apply-test-localhost: build
	./smoke_apply_local.sh

smoke-apply-bastion-test: build
	./smoke_apply_bastion.sh

smoke-apply-forward-test: build
	./smoke_apply_forward.sh

smoke-upgrade-test: build
	./smoke_upgrade.sh

smoke-prune-test: build
	./smoke_prune.sh

smoke-reset-test: build
	./smoke_reset.sh

smoke-cleanup:
	./smoke_cleanup.sh

smoke-test: smoke-apply-test
