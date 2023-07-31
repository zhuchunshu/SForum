#!/bin/bash

# 使用find命令获取./app/Plugins/*/src/目录下的所有一级子目录
directories=$(find ./app/Plugins/*/src -maxdepth 1 -type d)

# 遍历每个子目录并输出路径和文件夹名（当文件夹名不等于"src"或"migrations"时）
for dir in $directories; do
  # 获取文件夹名
  folder_name=$(basename "$dir")

  # 检查文件夹名是否不等于"src"或"migrations"
  if [ "$folder_name" != "src" ] && [ "$folder_name" != "migrations" ]; then
    # 输出路径和文件夹名
    php vendor/bin/regenerate-models.php "$dir"
  fi
done
