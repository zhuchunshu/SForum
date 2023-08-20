<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Handler\Store;

// 本地存储
use App\Plugins\Core\src\Annotation\FileStoreAnnotation;
use App\Plugins\Core\src\Handler\FileStoreInterface;
use App\Plugins\Core\src\Handler\Store\Handler\LocalStoreHandler;
use Hyperf\HttpMessage\Upload\UploadedFile;
use Hyperf\Stringable\Str;
use SplFileInfo;
use Swoole\Coroutine\System;
#[FileStoreAnnotation]
class LocalFileStore implements FileStoreInterface
{
    public function name() : string
    {
        return '本地存储';
    }
    public function handler() : string
    {
        return LocalStoreHandler::class;
    }
    public function view() : string
    {
        return 'User::Admin.FileStore.local';
    }
    public function save(UploadedFile $file, $folder, $file_prefix = null, $move = false, $path = null)
    {
        // 移动文件
        if ($move === true && file_exists($path)) {
            $filename = (new SplFileInfo($path))->getFilename();
            $folder_name = "upload/{$folder}/" . date('Ym/d', time());
            if (!is_dir(public_path($folder_name))) {
                if (!mkdir($concurrentDirectory = public_path($folder_name), 0777, true) && !is_dir($concurrentDirectory)) {
                    throw new \RuntimeException(sprintf('Directory "%s" was not created', $concurrentDirectory));
                }
            }
            System::exec('mv ' . $path . ' ' . public_path($folder_name . "/" . $filename));
            return ['url' => "/{$folder_name}/{$filename}", 'path' => public_path("{$folder_name}/{$filename}"), 'success' => true];
        }
        // 从零处理文件
        if (!$file_prefix) {
            $file_prefix = Str::random();
        }
        $folder_name = "upload/{$folder}/" . date('Ym/d', time());
        if (!is_dir(public_path($folder_name))) {
            if (!mkdir($concurrentDirectory = public_path($folder_name), 0777, true) && !is_dir($concurrentDirectory)) {
                throw new \RuntimeException(sprintf('Directory "%s" was not created', $concurrentDirectory));
            }
        }
        // 获取后缀名
        $extension = strtolower(@$file->getExtension()) ?: 'png';
        // 拼接文件名
        $filename = $file_prefix . '_' . time() . '_' . Str::random(10) . '.' . $extension;
        // 将图片移动到我们的目标存储路径中
        $file->moveTo(public_path($folder_name . '/' . $filename));
        return ['url' => "/{$folder_name}/{$filename}", 'path' => public_path("{$folder_name}/{$filename}"), 'success' => true];
    }
}