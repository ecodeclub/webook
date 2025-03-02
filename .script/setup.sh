# Copyright 2021 ecodeclub
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SOURCE_COMMIT=.github/pre-commit
TARGET_COMMIT=.git/hooks/pre-commit
SOURCE_PUSH=.github/pre-push
TARGET_PUSH=.git/hooks/pre-push

echo "设置 git pre-commit hooks..."
cp -f $SOURCE_COMMIT $TARGET_COMMIT

echo "设置 git pre-push hooks..."
cp -f $SOURCE_PUSH $TARGET_PUSH

# add permission to TARGET_PUSH and TARGET_COMMIT file.
test -x $TARGET_PUSH || chmod +x $TARGET_PUSH
test -x $TARGET_COMMIT || chmod +x $TARGET_COMMIT

echo "安装 golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0

echo "安装 goimports..."
go install golang.org/x/tools/cmd/goimports@latest

echo "安装 mockgen..."
go install go.uber.org/mock/mockgen@latest

echo "安装 wire..."
go install github.com/google/wire/cmd/wire@latest