<?php
namespace App\CodeFec;

use App\Model\AdminPlugin;


class CodeFec {

    public function handle(){
        $this->menu();
        $this->header();
        $this->boot();
        $this->plugins();
    }

    // 注册菜单
    public function menu(){

        require BASE_PATH."/app/CodeFec/Menu/default.php";

    }

    //创建页头内容
    public function header(){
        require BASE_PATH."/app/CodeFec/Header/default.php";
    }

    public function boot(){
        require BASE_PATH."/app/CodeFec/bootstrap.php";
    }
    // 处理插件
    public function plugins(){
        $array = AdminPlugin::query()->where("status",1)->get();
        $result = [];
        foreach ($array as $value) {
            $result[]=$value->name;
        }
        foreach ($result as $value) {
            if(file_exists(plugin_path($value."/".$value.".php"))){
                $class = "\App\Plugins\\".$value."\\".$value;
                if(@method_exists(new $class(),"handle")){
                    (new $class())->handle();
                }
            }
        }
    }

}