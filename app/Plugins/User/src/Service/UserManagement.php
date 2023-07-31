<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Service;

use App\Plugins\User\src\Annotation\UserManagementAnnotation;
use App\Plugins\User\src\Service\interfaces\UMHandlerInterface;
use App\Plugins\User\src\Service\interfaces\UserManagementInterface;
use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\Di\Annotation\Inject;
/**
 * 用户管理.
 */
class UserManagement
{
    /**
     * @var AnnotationCollector
     */
    #[Inject]
    protected AnnotationCollector $AnnotationCollector;
    /*
     * 获取所有管理接口
     */
    public function get_all_interface() : array
    {
        $arr = [];
        $all = $this->AnnotationCollector::getClassesByAnnotation(UserManagementAnnotation::class) ?: [];
        $all = array_keys($all);
        foreach ($all as $item) {
            if (new $item() instanceof UserManagementInterface) {
                $arr[] = $item;
            }
        }
        return $arr;
    }
    // 获取所有管理接口,带信息
    public function get_all() : array
    {
        $all = [];
        foreach ($this->get_all_interface() as $item) {
            $obj = new $item();
            $data = [];
            $data['show_view'] = $obj->show_view();
            $data['edit_view'] = $obj->edit_view();
            $data['handler'] = $obj->handler();
            $all[md5($item)] = $data;
        }
        return $all;
    }
    // 获取所有处理器
    public function get_all_handler() : array
    {
        $all = [];
        foreach ($this->get_all() as $data) {
            if (@new $data['handler']() instanceof UMHandlerInterface) {
                $all[] = $data['handler'];
            }
        }
        return $all;
    }
}