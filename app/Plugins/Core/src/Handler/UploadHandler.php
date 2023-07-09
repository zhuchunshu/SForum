<?php

declare(strict_types=1);
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
use Illuminate\Support\Str;
use Intervention\Image\ImageManagerStatic as Image;

// 上传图片
class UploadHandler
{
    public function save($file, $folder, $file_prefix = null, $max_width = 1500): array
    {
        if (! auth()->check() && ! admin_auth()->Check()) {
            return [
                'path' => '/404.jpg',
                'success' => false,
                'status' => '上传失败:未登录',
            ];
        }
        if (! $file_prefix) {
            $file_prefix = Str::random();
        }
        $folder_name = "upload/{$folder}/" . date('Ym/d', time());
        if (! is_dir(public_path($folder_name))) {
            mkdir(public_path($folder_name), 0777, true);
        }
        $upload_path = public_path() . '/' . $folder_name;
        $extension = strtolower($file->getExtension()) ?: 'png';
        $random = Str::random(10);
        $filename = $file_prefix . '_' . time() . '_' . $random . '.' . $extension;
        $_filename = $file_prefix . '_' . time() . '_' . $random;
        $file->moveTo(public_path($folder_name . '/' . $filename));
        $path = public_path("{$folder_name}/{$filename}");
        if ($max_width && $extension !== 'gif') {
            // 此类中封装的函数，用于裁剪图片
            $this->reduceSize($upload_path . '/' . $filename, $max_width);
            $to = public_path("{$folder_name}/{$_filename}.webp");
            $this->webp($path, $to);
            $path = $to;
        }

        $service = new FileStoreService();
        $upload = $service->save($file, $folder, $file_prefix, true, $path);
        if ($upload['success'] !== true) {
            return [
                'path' => 'error',
                'success' => false,
                'status' => '上传失败',
            ];
        }
        // 上传成功

        if (auth()->check()) {
            // 已登陆
            UserUpload::query()->create([
                'user_id' => auth()->id(),
                'path' => $upload['path'],
                'url' => $upload['url'],
            ]);
        }
        return [
            'path' => $upload['url'],
            'raw_path' => $upload['path'],
            'success' => $upload['success'],
            'status' => '上传成功!',
        ];
    }

    public function reduceSize($file_path, $max_width)
    {
        // 先实例化，传参是文件的磁盘物理路径
        $image = Image::make($file_path);

        // 进行大小调整的操作
        $image->resize($max_width, null, function ($constraint) {
            // 设定宽度是 $max_width，高度等比例缩放
            $constraint->aspectRatio();

            // 防止裁图时图片尺寸变大
            $constraint->upsize();
        });

        // 对图片修改后进行保存
        $image->save();
    }

    private function webp($from, $to)
    {
        $image = Image::make($from);
        $image->encode('webp');
        $image->save($to);
        unlink($from);
    }
}
