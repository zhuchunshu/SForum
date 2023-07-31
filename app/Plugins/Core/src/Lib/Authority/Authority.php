<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib\Authority;

// 权限管理模块
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
class Authority
{
    // action列表
    public function all() : array
    {
        return Itf()->get('core_auth');
    }
    // 新增action
    public function add(string $name, string $description = null)
    {
        Itf()->add('core_auth', $name, ['description' => $description, 'name' => $name]);
    }
    // action列表
    public function get() : array
    {
        $arr = [];
        foreach (Authority()->all() as $value) {
            $arr[] = $value;
        }
        return $arr;
    }
    public function check($quanxian) : bool
    {
        if (!auth()->check()) {
            return false;
        }
        $userClassData = UserClass::query()->where('id', auth()->data()->class_id)->first();
        $data = json_decode($userClassData['quanxian'], true);
        return in_array($quanxian, $data, true);
    }
    public function checkUser($quanxian, $user_id) : bool
    {
        if (!User::query()->where('id', $user_id)->exists()) {
            return false;
        }
        $user = User::query()->find($user_id);
        $userClassData = UserClass::query()->where('id', $user->class_id)->first();
        $data = json_decode($userClassData['quanxian'], true);
        return in_array($quanxian, $data, true);
    }
    public function getName(string $quanxian) : string|null
    {
        $name = null;
        foreach (Authority()->get() as $value) {
            if ($value['name'] === $quanxian) {
                $name = $value['description'];
            }
        }
        return $name;
    }
    // 获取拥有相关权限的所有用户
    public function getUsers(string $quanxian)
    {
        $userClassIds = [];
        foreach (UserClass::query()->get() as $value) {
            $data = json_decode($value->quanxian, true);
            if (in_array($quanxian, $data)) {
                $userClassIds[] = $value->id;
            }
        }
        $users = [];
        foreach (User::query()->get() as $value) {
            if (in_array($value->class_id, $userClassIds)) {
                $users[$value->id] = $value;
            }
        }
        return $users;
    }
}