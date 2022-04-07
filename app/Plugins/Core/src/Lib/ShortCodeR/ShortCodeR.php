<?php


namespace App\Plugins\Core\src\Lib\ShortCodeR;


use Hyperf\Di\Annotation\AnnotationCollector;
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
	    $arr = Itf()->get("ShortCodeR");
	    $shortCodeR = AnnotationCollector::getMethodsByAnnotation(\App\CodeFec\Annotation\ShortCode\ShortCodeR::class);
	    foreach ($shortCodeR as $data){
		    $name = $data['annotation']->name;
		    $callback = $data['class']."@".$data['method'];
		    $arr["ShortCodeR_".$name]=['callback' => $callback];
	    }
	    return $arr;
    }

    public function get($tag):bool|array{
        if($this->has($tag)){
            return $this->all()[$tag];
        }
        return false;
    }

    public function has($tag):bool{
        if(Arr::has($this->all(),"ShortCodeR_".$tag)){
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
	
	public function filter_make($method,...$content)
	{
		return (new Filter())->$method(...$content);
	}

    public function handle($content){

        return $this->to($this->to($content));
    }
	
	
	
	public function filter($content){
		return $this->filter_to($this->filter_to($content));
	}

    public function to($content){
        $y = $content;
        $content = $this->make("default",$content);
        if($content ===$y){
            $content = $this->make("type1",$content);
        }
        if($content ===$y){
            $content = $this->make("type2",$content);
        }
        if($content ===$y){
            $content = $this->make("type4",$content);
        }
        return $content;
    }
	
	public function filter_to($content){
		$y = $content;
		$content = $this->filter_make("default",$content);
		if($content ===$y){
			$content = $this->filter_make("type1",$content);
		}
		if($content ===$y){
			$content = $this->filter_make("type2",$content);
		}
		if($content ===$y){
			$content = $this->filter_make("type4",$content);
		}
		return $content;
	}

    public function callback($callback,...$parameter){
        $class = Str::before($callback,"@");
        $method = Str::after($callback,"@");
        return (new $class())->$method(...$parameter);
    }
}