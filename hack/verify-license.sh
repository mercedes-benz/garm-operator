#!/bin/bash
# SPDX-License-Identifier: MIT

set -e

array_contains() {
    local search="$1"
    local element
    shift
    for element; do
        if [[ "${element}" == "${search}" ]]; then
            return 0
        fi
    done
    return 1
}

HEADER="SPDX-License-Identifier: MIT"

all_files=()
export IFS=$'\n'
while IFS='' read -r line; do all_error_files+=("$line"); done < <(git ls-files | grep -v -E '\.excalidraw$|\.jpeg$|\.jpg$|\.gif$|\.png$|\.yaml$|^(go.sum|LICENSE|PROJECT|hack/boilerplate.go.txt)$|.*zz_generated.deepcopy.go$|.*zz_generated.conversion.go$')
unset IFS

errors=()
for file in "${all_error_files[@]}"; do
    array_contains "$file" "${HEADER_WHITELIST[@]}" && in_whitelist=$? || in_whitelist=$?
    if [[ "${in_whitelist}" -eq "0" ]]; then
        continue
    fi
    set +e
    matches=$(head -n 2 $file | grep "$HEADER" | wc -l)
    set -e
    if [[ "$matches" -ne "1" ]]; then
        errors+=("${file}")
        echo "error checking ${file} for the SPDX header"
    fi
done

if [ ${#errors[@]} -eq 0 ]; then
    echo 'Congratulations! All source files have been checked for SPDX header.'
else
    echo
    echo 'Please review the above files. They seem to miss the following header as comment:'
    echo "$HEADER"
    echo
    exit 1
fi
