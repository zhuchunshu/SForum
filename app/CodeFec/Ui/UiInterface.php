<?php
namespace App\CodeFec\Ui;

interface UiInterface{
    /**
     * 获取页头菜单数组
     *
     * @return array
     */
    public function get();
    /**
     * 创建UI钩子
     *
     * @param integer 唯一id $id
     * @param integer 类型 $type (css,js,...)
     * @param string 值 $value
     * @return boolean
     */
    public function add(int $id,string $type,string $value);
}