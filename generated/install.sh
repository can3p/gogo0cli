#!/bin/sh
# Based in flyctl installer: Copyright 2023 flyctl authors.
# Based on Deno installer: Copyright 2019 the Deno authors. All rights reserved. MIT license.
# TODO(everyone): Keep this script simple and easily auditable.

set -e

main() {
	os=$(uname -s)
	arch=$(uname -m)

	# this is quite ugly, but this way we do not depend on the json parse, nor do we depend
	# on any particular formatting, just the value
	version=$(curl -s https://api.github.com/repos/can3p/gogo-cli/releases/latest | grep -o 'https://github.com/can3p/gogo-cli/releases/tag/v[0-9.]*' | grep -o 'v[0-9.]*$')

	if [ ! -z $1 ]; then
		version="v$1"
	fi

	release_uri="https://github.com/can3p/gogo-cli/releases/download/$version/gogo-cli_${os}_${arch}.tar.gz"
	echo "Getting version $version, $release_uri"

	install_path="${CUSTOM_INSTALL:-$HOME}"

	bin_dir="$install_path/bin"
	ts=$(date +%s)
	tmp_dir="$install_path/gogo-cli_tmp$ts"
	exe="$bin_dir/gogo-cli"

	mkdir -p "$bin_dir"
	mkdir -p "$tmp_dir"

	function cleanup {
		rm -rf $tmp_dir
	}
	# be a good citizen and clean up after yourself
	trap cleanup EXIT

	curl -q --fail --location --progress-bar --output "$tmp_dir/gogo-cli.tar.gz" "$release_uri"
	# extract to tmp dir so we don't open existing executable file for writing:
	tar -C "$tmp_dir" -xzf "$tmp_dir/gogo-cli.tar.gz"
	chmod +x "$tmp_dir/gogo-cli"
	# atomically rename into place:
	mv "$tmp_dir/gogo-cli" "$exe"
	rm "$tmp_dir/gogo-cli.tar.gz"

	echo "gogo-cli was installed successfully to $exe"
	if command -v gogo-cli >/dev/null; then
		echo "Run 'gogo-cli help' to get started"
	else
		case $SHELL in
		/bin/zsh) shell_profile=".zshrc" ;;
		*) shell_profile=".bash_profile" ;;
		esac
		echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
		echo "  export install_path=\"$install_path\""
		echo "  export PATH=\"\$install_path/bin:\$PATH\""
		echo "Run '$exe --help' to get started"
	fi
}

main "$1"
