#
# Copyright 2023.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

header() {
	local title="🔆🔆🔆  $*  🔆🔆🔆 "

	local len=40
	if [[ ${#title} -gt $len ]]; then
		len=${#title}
	fi

	echo -e "\n\n  \033[1m${title}\033[0m"
	echo -n "━━━━━"
	printf '━%.0s' $(seq "$len")
	echo "━━━━━━━"

}

info() {
	echo -e " 🔔 $*" >&2
}

err() {
	echo -e " 😱 $*" >&2
}

warn() {
	echo " ⚠️  $*" >&2
}

ok() {
	echo " ✅ $*" >&2
}

skip() {
	echo " 🙈 SKIP: $*" >&2
}

run() {
	echo -e " ❯ $*\n" >&2
	"$@"
}

die() {
	echo -e "\n ✋ $* "
	echo -e "──────────────────── ⛔️⛔️⛔️ ────────────────────────\n"
	exit 1
}

line() {
	local len="$1"
	shift

	echo -n "────" >&2
	printf '─%.0s' $(seq "$len") >&2
	echo "────────" >&2
}
