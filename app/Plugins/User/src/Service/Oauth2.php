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

use App\Plugins\User\src\Annotation\Oauth2Annotation;
use App\Plugins\User\src\Service\interfaces\Oauth2Interface;
use App\Plugins\User\src\Service\interfaces\Oauth2SettingInterface;
use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\Di\Annotation\Inject;
class Oauth2
{
    /**
     * @var AnnotationCollector
     */
    #[Inject]
    protected AnnotationCollector $AnnotationCollector;
    /*
     * 获取所有登陆接口
     */
    public function get_all_interface() : array
    {
        $arr = [];
        $all = $this->AnnotationCollector::getClassesByAnnotation(Oauth2Annotation::class) ?: [];
        $all = array_keys($all);
        foreach ($all as $item) {
            if (new $item() instanceof Oauth2Interface) {
                $arr[] = $item;
            }
        }
        return $arr;
    }
    /*
     * 获取所有登陆接口(带信息)
     */
    public function get_all() : array
    {
        $all = [];
        foreach ($this->get_all_interface() as $item) {
            $obj = new $item();
            $data = [];
            $data['mark'] = $obj->mark();
            $data['admin_name'] = $obj->name();
            $data['name'] = $obj->name();
            $data['icon'] = $obj->icon();
            $data['view'] = $obj->view();
            $data['admin_view'] = $obj->admin_view();
            $data['handler'] = $obj->setting_handler();
            $all[$obj->mark()] = $data;
        }
        return $all;
    }
    /*
     * 获取所有登陆接口(不带信息)
     */
    public function _get_all() : array
    {
        $all = [];
        foreach ($this->get_all_interface() as $item) {
            $obj = new $item();
            $all[] = $obj->mark();
        }
        return array_unique($all);
    }
    /**
     * 获取所有设置处理器.
     */
    public function get_all_handler() : array
    {
        $all = [];
        foreach ($this->get_all() as $data) {
            if (@new $data['handler']() instanceof Oauth2SettingInterface) {
                $all[] = $data['handler'];
            }
        }
        return $all;
    }
    /**
     * 获取所有已启用的接口.
     * @return array|mixed
     */
    public function get_enables()
    {
        if (!get_options('oauth2_enable')) {
            return [];
        }
        $result = json_decode(get_options('oauth2_enable'), true);
        $all = [];
        foreach ($result as $item) {
            if (in_array($item, $this->_get_all())) {
                $all[] = $item;
            }
        }
        return $all;
    }
    /**
     * 获取所有已启用的接口信息.
     */
    public function get_enables_data() : array
    {
        $data = [];
        foreach ($this->get_enables() as $value) {
            $data[$value] = $this->get_data($value);
        }
        return $data;
    }
    /**
     * 判断接口是否已启用.
     * @param mixed $mark
     */
    public function check_enable($mark) : bool
    {
        return in_array($mark, $this->get_enables());
    }
    /**
     * 获取单个接口信息.
     * @param $mark
     */
    public function get_data($mark) : array
    {
        if (!in_array($mark, $this->_get_all())) {
            return [];
        }
        return $this->get_all()[$mark];
    }
}
function oauth2() : Oauth2
{
    return new Oauth2();
}