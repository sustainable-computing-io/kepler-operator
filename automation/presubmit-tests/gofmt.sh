#!/usr/bin/env bash
#
# This file is part of the Kepler project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 The Kepler Contributors.
#

set -e

echo "Checking go format"
unformatted="$(gofmt -e -d -s -l "./cmd" "./pkg")"
[[ -z "$unformatted" ]] && exit 0

# Some files are not gofmt.
echo "The following Go files must be formatted with gofmt:" >&2
for fn in $unformatted; do
	echo "  $fn" >&2
done

echo "Please run 'make format'." >&2
exit 1
