<?php
namespace App\CodeFec;

use App\Model\AdminPlugin;


class CodeFec {

    public function handle(): void
    {
        $this->menu();
        $this->header();
        $this->boot();
        $this->plugins();
        $this->setting();
        $this->route();
        $this->itf();
    }

    public function setting(): void
    {
        require BASE_PATH."/app/CodeFec/Itf/Setting/default.php";
    }

    // 注册菜单
    public function menu(): void
    {

        require BASE_PATH."/app/CodeFec/Menu/default.php";

    }

    //创建页头内容
    public function header(): void
    {
        require BASE_PATH."/app/CodeFec/Header/default.php";
    }

    public function boot(): void
    {
        require BASE_PATH."/app/CodeFec/bootstrap.php";
    }

    /**
     * 重写路由
     */
    public function route(): void
    {
        require BASE_PATH."/app/CodeFec/Itf/Route/default.php";
    }

    // 处理插件
    public function plugins(): void
    {
        $result = (new Plugins())->getEnPlugins();
        foreach ($result as $value) {
            if(file_exists(plugin_path($value."/".$value.".php"))){
                $class = "\App\Plugins\\".$value."\\".$value;
                if(@method_exists(new $class(),"handler")){
                    (new $class())->handler();
                }
            }
        }
    }

    public function itf(): void
    {
        require BASE_PATH."/app/CodeFec/Itf/Itf/default.php";
    }

}