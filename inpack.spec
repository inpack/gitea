[project]
name = gitea
version = 1.10.0
vendor = gitea.io
homepage = https://gitea.io
groups = app/dev,app/prod,app/co
description = A painless self-hosted Git service fork of Gogs.


%build
PREFIX="/opt/gitea/gitea"

cd {{.inpack__pack_dir}}/deps

if [ ! -f "gitea-{{.project__version}}-linux-amd64" ]; then
    wget "https://dl.gitea.io/gitea/{{.project__version}}/gitea-{{.project__version}}-linux-amd64"
    strip -s gitea-{{.project__version}}-linux-amd64
fi

install gitea-{{.project__version}}-linux-amd64 {{.buildroot}}/gitea
chmod +x {{.buildroot}}/gitea

mkdir -p {{.buildroot}}/custom/conf
mkdir -p {{.buildroot}}/data/repos
mkdir -p {{.buildroot}}/var/log

%files
misc/

