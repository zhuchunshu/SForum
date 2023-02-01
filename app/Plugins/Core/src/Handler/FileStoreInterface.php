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

use Hyperf\HttpMessage\Upload\UploadedFile;

interface FileStoreInterface
{
    /*
     * 获取服务名称
     */
    public function name(): string;

    /**
     * 保存文件.
     * @param UploadedFile $file
     * @param $folder
     * @param null $file_prefix
     * @param mixed $move
     */
    public function save(UploadedFile $file, $folder, $file_prefix = null, $move = false,$path=null);

    /**
     * 后台处理器.
     */
    public function handler(): string;

    /**
     * 后台视图.
     */
    public function view(): string;
}
