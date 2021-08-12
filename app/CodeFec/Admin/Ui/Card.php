<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace  App\CodeFec\Admin\Ui;

use Illuminate\Support\Str;
use Psr\Http\Message\ResponseInterface;

/**
 * 创建一个卡片.
 */
class Card
{
    /**
     * 卡片标题.
     * @var string
     */
    public $title = '';

    /**
     * 卡片类型.
     * @var int
     */
    public $titleType = 0;

    /**
     * 卡片内容.
     */
    public $content;

    /**
     * 续增Class.
     */
    public $AddClass;

    public $id;

    /**
     * 设置卡片标题.
     * @param string $title
     * @return Card
     */
    public function Title(string $title): Card
    {
        $this->title = $title;
        return $this;
    }

    /**
     * 设置卡片标题类型.
     * @param int $type
     * @return Card
     */
    public function TitleType(int $type): Card
    {
        $this->titleType = $type;
        return $this;
    }

    /**
     * 设置卡片内容.
     * @param mixed $content
     */
    public function Content($content): Card
    {
        $this->content = $content;
        return $this;
    }

    /**
     * 续增Class.
     * @param string $class
     * @return Card
     */
    public function AddClass(string $class): Card
    {
        $this->AddClass = $class;
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

    /**
     * 渲染卡片.
     */
    public function render(): ResponseInterface
    {
        if(!$this->content){
            $this->content = "无内容";
        }
        if(!$this->id){
            $this->id = Str::random(7);
        }
        return view('admin.Ui.card', ['title' => $this->title, 'titleType' => $this->titleType, 'content' => $this->content, 'AddClass' => $this->AddClass,"id" => $this->id]);
    }
}
