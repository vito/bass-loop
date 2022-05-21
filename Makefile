nix/vendorSha256.txt: go.mod go.sum
	./hack/get-nix-vendorsha > $@
