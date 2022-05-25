<?php
namespace App\CodeFec\Admin\Ui;

use Illuminate\Support\Str;

class Row {

    public $row="col-md-12";

    public $content;

    public $id;
    
    public function row($row){
        $this->row = $row;
        return $this;
    }

    public function content($content){
        $this->content = $content;
        return $this;
    }

    public function id($id=""){
        if(!$id){
            $this->id = Str::random(7);
        }else{
            $this->id = $id;
        }
        return $this;
    }

    public function render(){
        if(!$this->content){
            $this->content = "无内容";
        }
        if(!$this->id){
            $this->id = Str::random(7);
        }
        return view("admin.Ui.row",["row"=>$this->row,"content" => $this->content,"id" => $this->id]);
    }

}