<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Handler;

use App\Plugins\Core\src\Service\FileStoreService;
use App\Plugins\User\src\Models\UserUpload;
class FileUpload
{
    public function save($file, $folder, $file_prefix = null) : array
    {
        if (!auth()->check() && !admin_auth()->Check()) {
            return ['path' => '/404.jpg', 'success' => false, 'status' => '上传失败:未登录'];
        }
        $service = new FileStoreService();
        $upload = $service->save($file, $folder, $file_prefix);
        if ($upload['success'] !== true) {
            return ['path' => 'error', 'success' => false, 'status' => '上传失败'];
        }
        // 上传成功
        if (auth()->check()) {
            // 已登陆
            UserUpload::query()->create(['user_id' => auth()->id(), 'path' => $upload['path'], 'url' => $upload['url']]);
        }
        return ['path' => $upload['url'], 'raw_path' => $upload['path'], 'success' => $upload['success'], 'status' => '上传成功!'];
    }
}