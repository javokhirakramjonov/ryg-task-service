update-protos:
	git submodule update --remote
	scripts/genProto.sh
