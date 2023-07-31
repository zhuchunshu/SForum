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
use Illuminate\Support\Str;
use Intervention\Image\ImageManagerStatic as Image;
class AvatarUpload
{
    public function save($file, $folder, $file_prefix = null, $max_width = false) : array
    {
        if (!auth()->check() && !admin_auth()->Check()) {
            return ['path' => '/404.jpg', 'success' => false, 'status' => '上传失败:未登录'];
        }
        if (!$file_prefix) {
            $file_prefix = Str::random();
        }
        $folder_name = "upload/{$folder}/" . date('Ym/d', time());
        if (!is_dir(public_path($folder_name))) {
            mkdir(public_path($folder_name), 0777, true);
        }
        $upload_path = public_path() . '/' . $folder_name;
        $extension = strtolower($file->getExtension()) ?: 'png';
        $random = Str::random(10);
        $filename = $file_prefix . '_' . time() . '_' . $random . '.' . $extension;
        $_filename = $file_prefix . '_' . time() . '_' . $random;
        $file->moveTo(public_path($folder_name . '/' . $filename));
        $path = public_path("{$folder_name}/{$filename}");
        // 检查是否允许用户设置 GIF 头像，并且文件扩展名是 GIF
        if (get_options('user_set_avatar_gif') === 'true' && $extension === 'gif') {
            // 如果允许 GIF 头像，直接跳过处理
        } elseif ($max_width && $extension !== 'webp') {
            // 如果有最大宽度限制，并且文件扩展名不是 webp
            // 调用 reduceSize 函数，用于裁剪图片并调整大小
            $this->reduceSize($upload_path . '/' . $filename, $max_width);
            // 将裁剪后的图片转换为 webp 格式
            $to = public_path("{$folder_name}/{$_filename}.webp");
            $this->webp($path, $to);
            // 更新路径为转换后的 webp 图片路径
            $path = $to;
        }
        $service = new FileStoreService();
        $upload = $service->save($file, $folder, $file_prefix, true, $path);
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