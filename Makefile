prefix = /opt
srvprefix = /srv
varprefix = /var/opt

srvdir = $(DESTDIR)$(srvprefix)/plantstation
servicedir = $(DESTDIR)/etc/systemd/system

srcdir = .
outdir = .

program = plantstation

files = $(DESTDIR)$(prefix)/bin/$(program)
files += $(patsubst $(srcdir)/service/%,$(DESTDIR)$(prefix)/bin/%,$(wildcard $(srcdir)/service/*.sh))
files += $(patsubst $(srcdir)/service/%,$(servicedir)/%,$(wildcard $(srcdir)/service/*.service $(srcdir)/service/*.path))
files += $(DESTDIR)$(varprefix)/plantstation
files += $(patsubst $(srcdir)/camweb/%,$(srvdir)/camweb/%,$(wildcard $(srcdir)/camweb/*.html))
files += $(patsubst $(srcdir)/web/%,$(srvdir)/web/%,$(wildcard $(srcdir)/web/*.html))
files += $(patsubst $(srcdir)/web/js/%,$(srvdir)/web/js/%,$(wildcard $(srcdir)/web/js/*.js))
files += $(DESTDIR)$(varprefix)/.well-known/acme-challenge
files += $(srvdir)/acme-challenge

all: $(outdir)/$(program)

$(outdir)/$(program): $(wildcard $(srcdir)/*.go)
	GOARM=6 GOARCH=arm GOOS=linux go build -o $@

install: $(files)

$(DESTDIR)$(prefix)/bin/%: $(outdir)/%
	install -DTm755 $< $@

$(DESTDIR)$(varprefix)/plantstation:
	install -d $@

# create link for certbot in writeable filesystem
$(DESTDIR)$(varprefix)/.well-known/acme-challenge:
	install -d $@

# create link in readonly resource path to writeable folder
$(srvdir)/acme-challenge:
	ln -snf $(varprefix)/.well-known/acme-challenge $@

$(DESTDIR)$(prefix)/bin/%.sh: $(srcdir)/service/%.sh
	install -DTm755 $< $@

$(servicedir)/%.service: $(srcdir)/service/%.service
	install -DTm644 $< $@

$(servicedir)/%.path: $(srcdir)/service/%.path
	install -DTm644 $< $@

$(DESTDIR)$(srvprefix)/plantstation/camweb/%.html: $(srcdir)/camweb/%.html
	install -DTm600 $< $@

$(DESTDIR)$(srvprefix)/plantstation/web/%.html: $(srcdir)/web/%.html
	install -DTm600 $< $@

$(DESTDIR)$(srvprefix)/plantstation/web/js/%.js: $(srcdir)/web/js/%.js
	install -DTm600 $< $@

clean:
	rm $(outdir)/$(program)
