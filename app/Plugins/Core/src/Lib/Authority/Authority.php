<?php

namespace App\Plugins\Core\src\Lib\Authority;

// 权限管理模块
use App\Plugins\User\src\Models\UserClass;
use Hyperf\Utils\Arr;

class Authority
{
    // action列表
    public function all(): array
    {
        return Itf()->get("core_auth");
    }

    // 新增action
    public function add(string $name,string $description=null)
    {
        Itf()->add("core_auth",$name,[
            "description" => $description,
            "name" => $name,
        ]);
    }

    // action列表
    public function get(): array
    {
        $arr = [];
        foreach(Authority()->all() as $value){
            $arr[]=$value;
        }
        return $arr;
    }

    public function check($quanxian):bool{
        if(!auth()->check()){
           return false;
        }
        $userClassData = UserClass::query()->where("id",auth()->data()->class_id)->first();
        $data = json_decode($userClassData['quanxian'],true);
        return in_array($quanxian, $data,true);
    }

}