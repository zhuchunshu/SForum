<?php


namespace App\Plugins\Core\src\Lib\ShortCodeR;


use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;

class ShortCodeR
{

    public function add($tag,$callback): void
    {
        Itf()->add("ShortCodeR",$tag,["callback" => $callback]);
    }

    public function all(): array
    {
        return Itf()->get("ShortCodeR");
    }

    public function get($tag):bool|array{
        if($this->has($tag)){
            return Itf()->get("ShortCodeR")[$tag];
        }
        return false;
    }

    public function has($tag):bool{
        if(Arr::has(Itf()->get("ShortCodeR"),"ShortCodeR_".$tag)){
            return true;
        }
        return false;
    }

    public function make_tag($tag): array{
        $start = "[".$tag."]";
        $end = "[".$tag."/]";
        return [
            "start" => $start,
            "end" => $end,
        ];
    }


    public function make($method,...$content)
    {
        return (new Make())->$method(...$content);
    }

    public function handle($content){
        $y = $content;
        $content = $this->make("default",$content);
        if($content ===$y){
            $content = $this->make("type1",$content);
        }
        if($content ===$y){
            $content = $this->make("type2",$content);
        }
        if($content ===$y){
            $content = $this->make("type3",$content);
        }
        if($content ===$y){
            $content = $this->make("type4",$content);
        }
        return $content;
    }

    public function callback($callback,...$parameter){
        $class = Str::before($callback,"@");
        $method = Str::after($callback,"@");
        return (new $class())->$method(...$parameter);
    }
}