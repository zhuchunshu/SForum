<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core;

use Hyperf\Collection\Arr;
use Swoole\Coroutine\System;

class Menu
{
    /**
     * handle.
     * @throws \RedisException
     * @return array
     */
    public function get()
    {
        $menu = [];
        foreach ($this->db() as $id => $data) {
            $data['id'] = (int) $id;
            $menu[$id] = $data;
        }
        $keys = array_keys($menu);
        $news_key = array_column($menu, 'sort');
        array_multisort($news_key, SORT_ASC, $menu, $keys);
        return array_combine($keys, $menu);
    }

    /**
     * get all keys.
     * @throws \RedisException
     * @return int[]|string[]
     */
    public function get_keys()
    {
        return array_keys($this->get());
    }

    /**
     * get menu data.
     * @param mixed $id
     * @throws \RedisException
     * @return null|mixed
     */
    public function get_data($id)
    {
        $id = (int) $id;
        if (! in_array($id, $this->get_keys())) {
            return null;
        }
        return $this->get()[$id];
    }

    /**
     * get all Itf menu.
     */
    private function Itf(): array
    {
        $menu = [];
        foreach (Itf()->get('menu') as $k => $value) {
            $k = core_Itf_id('menu', $k);
            if (Arr::has($value, 'quanxian') && $value['quanxian'] instanceof \Closure) {
                $value['quanxian'] = $this->serialize($value['quanxian']);
            }
            $value['Itf'] = true;
            $menu[$k] = $value;
        }
        return $menu;
    }

    /**
     * get all database menu.
     * @throws \RedisException
     */
    private function db(): array
    {
        $prefix_name = \Hyperf\Config\config('cache.default.prefix') . 'menus';
        foreach ($this->Itf() as $id => $item) {
            if (! redis()->hExists($prefix_name, (string) $id)) {
                redis()->hSetNx($prefix_name, (string) $id, $this->serialize($item));
            }
        }

        Itf()->_del('menu');
        $all = redis()->hGetAll($prefix_name);
        $result = [];
        foreach ($all as $id => $data) {
            $_data = unserialize($data);
            if (Arr::has($_data, 'quanxian')) {
                $_data['quanxian'] = _unserialize($_data['quanxian']);
            }
            if (! arr_has($_data, 'sort')) {
                $_data['sort'] = $id;
            }
            if (! arr_has($_data, 'hidden')) {
                $_data['hidden'] = false;
            }
            $result[(int) $id] = $_data;
        }
        return $result;
    }

    /**
     * @param $data
     * @return string
     */
    public function serialize($data): string
    {
        if(is_array($data)){
            foreach ($data as  $key => $item) {
                if(is_array($item)){
                    $data[$key]=$this->serialize($item);
                }else{
                    if($item instanceof \Closure){
                        $data[$key] = _serialize($item);
                    }else{
                        $data[$key] = $item;
                    }
                }
            }
            return serialize($data);
        }
        return _serialize($data);
    }

    /**
     * @param $data
     * @return mixed
     */
    private function unserialize($data)
    {
        return _unserialize($data);
    }


    public function backup($name='menu')
    {

        $data = [];
        foreach ($this->get() as $k=>$v){
            if(arr_has($v,'quanxian') && $v['quanxian'] instanceof \Closure){
                $v['quanxian']=$this->serialize($v['quanxian']);
            }
            $data[$k]=$v;
        }
        $menu_serialize =  $this->serialize($data);
        $menu = json_encode($data,JSON_PRETTY_PRINT,JSON_UNESCAPED_UNICODE);
        if (! is_dir(BASE_PATH . '/runtime/backup')) {
            System::exec('cd ' . BASE_PATH . '/runtime' . '&& mkdir ' . 'backup');
        }
        if (! is_dir(BASE_PATH . '/runtime/backup/menu')) {
            System::exec('cd ' . BASE_PATH . '/runtime/backup' . '&& mkdir ' . 'menu');
        }
        file_put_contents(BASE_PATH."/runtime/backup/menu/".$name."_serialize.txt",$menu_serialize);
        file_put_contents(BASE_PATH."/runtime/backup/menu/".$name.".json",$menu);
        return true;
    }

    /**
     * 导入
     */
    public function import(array $data,$recover=false){
        $prefix_name = config('cache.default.prefix') . 'menu';
        if($recover===true){
            redis()->del($prefix_name);
        }
        foreach ($data as$item) {
            if (! redis()->hExists($prefix_name, (string) $item['id'])) {
                redis()->hSetNx($prefix_name, (string) $item['id'], $this->serialize($item));
            }
        }
    }
}
