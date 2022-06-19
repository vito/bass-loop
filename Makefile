targets=html/base16-options.tmpl public/css/main.css

all: $(targets)

nix/vendorSha256.txt: go.mod go.sum
	./hack/get-nix-vendorsha > $@

view/Base16Options.svelte: public/css/base16/*.css hack/base16-tmpl
	./hack/base16-tmpl

clean:
	rm -f $(targets)
