<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Service;

// 文件存储服务
use App\Plugins\Core\src\Annotation\FileStoreAnnotation;
use App\Plugins\Core\src\Handler\FileStoreInterface;
use Hyperf\Di\Annotation\AnnotationCollector;

class FileStoreService
{
    // 获取所有储存服务
    public function get_services()
    {
        $services = [];
        foreach (AnnotationCollector::getClassesByAnnotation(FileStoreAnnotation::class) as $key => $value) {
            if (new $key() instanceof FileStoreInterface) {
                $services[md5($key)] = [
                    'name' => (new $key())->name(),
                    'handler' => (new $key())->handler(),
                    'view' => (new $key())->view(),
                    'class' => $key,
                ];
            }
        }
        return $services;
    }

    // 获取所有储存服务
    public function get_handlers()
    {
        $handlers = [];
        foreach (AnnotationCollector::getClassesByAnnotation(FileStoreAnnotation::class) as $key => $value) {
            if (new $key() instanceof FileStoreInterface) {
                $handlers[] = (new $key())->handler();
            }
        }
        return $handlers;
    }

    /**
     * 保存文件.
     * @param $file
     * @param $folder
     * @param null $file_prefix
     * @param mixed $move
     * @param null|mixed $path
     */
    public function save($file, $folder, $file_prefix = null, $move = false, $path = null): array
    {
        $file_store_service = get_options('file_store_service', 'b53a68eae9ecac0d86eb8d1125524b13');

        if (! arr_has($this->get_services(), $file_store_service)) {
            $file_store_service = key($this->get_services());
        }

        // 判断存储服务是否存在
        if ($file_store_service || arr_has($this->get_services(), $file_store_service)) {
            $services = $this->get_services();
            if (isset($services[$file_store_service])) {
                $class = $services[$file_store_service]['class'];
                $file_store = new $class();
                if ($file_store instanceof FileStoreInterface) {
                    return $file_store->save($file, $folder, $file_prefix, $move, $path);
                }
            }
        }
        return $this->response();
    }

    public function response($success = false, $file_path = null, $file_url = null)
    {
        return [
            'path' => $file_path,
            'url' => $file_url,
            'success' => $success,
        ];
    }
}
