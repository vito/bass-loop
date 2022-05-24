targets=nix/vendorSha256.txt html/base16-options.tmpl css/main.css

all: $(targets)

nix/vendorSha256.txt: go.mod go.sum
	./hack/get-nix-vendorsha > $@

html/base16-options.tmpl: css/base16/*.css hack/base16-tmpl
	./hack/base16-tmpl

css/main.css: less/main.less less/*.less
	yarn run lessc $< $@

clean:
	rm -f $(targets)
