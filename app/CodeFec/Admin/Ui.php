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
namespace App\CodeFec\Admin;

class Ui
{
    /**
     * 标题.
     *
     * @var string
     */
    public $title = 'Home';

    /**
     * 页头右侧按钮数组.
     *
     * @var array
     */
    public $headerBtn = [];

    /**
     * Body内容.
     */
    public $body = '无内容';

    /**
     * 导入js.
     *
     * @var array
     */
    public $JsUrl = [];

    /**
     * 导入css.
     *
     * @var array
     */
    public $CssUrl = [];

    /**
     * 设置标题.
     * @param string $title
     * @return Ui
     */
    public function title(string $title): Ui
    {
        $this->title = $title;
        return $this;
    }

    /**
     * 设置body内容.
     * @param mixed $body
     * @return Ui
     */
    public function body($body): Ui
    {
        $this->body = $body;
        return $this;
    }

    /**
     * 设置页头按钮.
     * @param array $array
     * @return Ui
     */
    public function headerBtn(array $array): Ui
    {
        $this->headerBtn = $array;
        return $this;
    }

    /**
     * 导入自定义js.
     * @param array $JsUrl
     * @return Ui
     */
    public function ImportJs(array $JsUrl): Ui
    {
        $this->JsUrl = $JsUrl;
        return $this;
    }

    /**
     * 导入自定义Css.
     * @param array $CssUrl
     * @return Ui
     */
    public function ImportCss(array $CssUrl): Ui
    {
        $this->CssUrl = $CssUrl;
        return $this;
    }

    /**
     * 渲染模板
     */
    public function render(): \Psr\Http\Message\ResponseInterface
    {
        return view('admin.ui', ['title' => $this->title, 'body' => $this->body, 'headerBtn' => $this->headerBtn, 'JsUrl' => $this->JsUrl, 'CssUrl' => $this->CssUrl]);
    }
}
