<?php

/**
 * 你无需读懂 我的代码
 */
namespace App\Plugins\Core\src\Lib\ShortCodeR;

use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\Collection\Arr;
use Hyperf\Stringable\Str;
use JetBrains\PhpStorm\Pure;
use Thunder\Shortcode\HandlerContainer\HandlerContainer;
use Thunder\Shortcode\Parser\RegularParser;
use Thunder\Shortcode\Processor\Processor;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;
class ShortCodeR
{
    /**
     * ShortCode 处理器
     * @var HandlerContainer
     */
    private HandlerContainer $handlers;
    /**
     * 不解析的ShortCode
     * @var array
     */
    private array $remove = [];
    public function __construct()
    {
        $this->handlers = new HandlerContainer();
    }
    /**
     * 设置删除的ShortCode
     * @param array $shortCodes
     * @return ShortCodeR
     */
    public function setRemove(array $shortCodes) : ShortCodeR
    {
        $this->remove = $shortCodes;
        return $this;
    }
    public function add($tag, $callback) : void
    {
        Itf()->add("ShortCodeR", $tag, ["callback" => $callback]);
    }
    /**
     * 获取全部ShortCode
     * @return array
     */
    public function all() : array
    {
        $arr = Itf()->get("ShortCodeR");
        $shortCodeR = AnnotationCollector::getMethodsByAnnotation(\App\CodeFec\Annotation\ShortCode\ShortCodeR::class);
        foreach ($shortCodeR as $data) {
            $name = $data['annotation']->name;
            $callback = $data['class'] . "@" . $data['method'];
            $arr["ShortCodeR_" . $name] = ['callback' => $callback];
        }
        // 去重
        $data = $this->unique($arr);
        // 清理不解析的ShortCode
        if (count($this->remove)) {
            $data = $this->remove($data);
        }
        return $data;
    }
    public function get($tag) : bool|array
    {
        if ($this->has($tag)) {
            return $this->all()[$tag];
        }
        return false;
    }
    public function has($tag) : bool
    {
        if (Arr::has($this->all(), "ShortCodeR_" . $tag)) {
            return true;
        }
        return false;
    }
    public function make_tag($tag) : array
    {
        $start = "[" . $tag . "]";
        $end = "[" . $tag . "/]";
        return ["start" => $start, "end" => $end];
    }
    public function make($method, $all, $content)
    {
        return (new Make())->{$method}($all, $content);
    }
    /**
     * 处理器
     * @param $content
     * @param $data
     * @return string
     */
    public function handle($content, $data)
    {
        $shortCodes = $this->all();
        foreach ($shortCodes as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $this->handlers->add($tag, function (ShortcodeInterface $s) use($value, $data) {
                $match = [$s->getContent(), $s->getContent(), $s->getContent()];
                return $this->callback($value['callback'], $match, $s, $data);
            });
        }
        $processor = new Processor(new RegularParser(), $this->handlers);
        return $processor->process($content);
    }
    /**
     * 筛选
     * @param $content
     * @return string
     */
    public function filter($content) : string
    {
        foreach ($this->all() as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $this->handlers->add($tag, function (ShortcodeInterface $s) use($value) {
                return '[' . $s->getName() . "]" . "安全性问题,禁止预览此标签内容[/" . $s->getName() . "]";
            });
        }
        $processor = new Processor(new RegularParser(), $this->handlers);
        return $processor->process($content);
    }
    /**
     * 回调
     * @param $callback
     * @param ...$parameter
     * @return mixed
     */
    public function callback($callback, ...$parameter) : mixed
    {
        $class = Str::before($callback, "@");
        $method = Str::after($callback, "@");
        return call_user_func_array([new $class(), $method], $parameter);
    }
    private function _() : array
    {
        $all = [];
        foreach (Itf()->get('_ShortCodeR') as $value) {
            $all[] = "ShortCodeR_" . $value;
        }
        return array_unique($all);
    }
    /**
     * 删除指定ShortCode
     * @param $data
     * @return array
     */
    private function remove($data) : array
    {
        $arr = [];
        foreach ($data as $k => $v) {
            if (!in_array($k, $this->_remove(), true)) {
                $arr[$k] = $v;
            }
        }
        return $arr;
    }
    /**
     * 被删除的ShortCode
     * @return array
     */
    private function _remove() : array
    {
        $all = [];
        foreach ($this->remove as $value) {
            $all[] = "ShortCodeR_" . $value;
        }
        return array_unique($all);
    }
    /**
     * 去重
     * @param array $data
     * @return array
     */
    private function unique(array $data)
    {
        $arr = [];
        foreach ($data as $k => $v) {
            if (!in_array($k, $this->_(), true)) {
                $arr[$k] = $v;
            }
        }
        return $arr;
    }
}