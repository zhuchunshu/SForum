<?php


namespace App\Plugins\Core\src\Lib\ShortCode;


use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;

class ShortCode
{

    public function add($tag,$callback): void
    {
        Itf()->add("ShortCode",$tag,["callback" => $callback]);
    }

    public function all(): array
    {
        return Itf()->get("ShortCode");
    }

    public function get($tag):bool|array{
        if($this->has($tag)){
            return Itf()->get("ShortCode")[$tag];
        }
        return false;
    }

    public function has($tag):bool{
        if(Arr::has(Itf()->get("ShortCode"),$tag)){
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


    public function make(): Make
    {
        return (new Make());
    }

    public function handle(string $content):string{
        $content = $this->make()->default($content);
        $content = $this->make()->type1($content);
        $content = $this->make()->type2($content);
        return $content;
    }

    public function callback($callback,...$parameter){
        $class = Str::before($callback,"@");
        $method = Str::after($callback,"@");
        return (new $class())->$method(...$parameter);
    }
}