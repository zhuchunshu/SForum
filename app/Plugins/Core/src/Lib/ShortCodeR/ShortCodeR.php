<?php
/**
 * 你无需读懂 我的代码
 */
namespace App\Plugins\Core\src\Lib\ShortCodeR;


use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;

class ShortCodeR
{
	public bool $comment  = false;
	
	public function comment($diff=true): ShortCodeR
	{
		$this->comment = $diff;
		return $this;
	}
	
	public bool $topic  = false;
	
	public function topic($diff=true): ShortCodeR
	{
		$this->topic = $diff;
		return $this;
	}

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
		$data = $this->diff($arr);
	    if($this->comment===true){
			$data = $this->diff_comment($data);
	    }
	    if($this->topic===true){
		    $data = $this->diff_topic($data);
	    }
		return $data;
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


    public function make($method,$all,$content)
    {
        return (new Make())->$method($all,$content);
    }
	
	public function filter_make($method,...$content)
	{
		return (new Filter())->$method(...$content);
	}

    public function handle($content){
        return $this->to($this->all(),$this->to($this->all(),$content));
    }
	
	public function filter($content){
		return $this->filter_to($this->filter_to($content));
	}

    public function to($all,$content){
        $y = $content;
        $content = $this->make("default",$all,$content);
        if($content ===$y){
            $content = $this->make("type1",$all,$content);
        }
        if($content ===$y){
            $content = $this->make("type2",$all,$content);
        }
        if($content ===$y){
            $content = $this->make("type4",$all,$content);
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
	
	private function _(): array
	{
		$all = [];
		foreach(Itf()->get('_ShortCodeR') as $value){
			$all[]="ShortCodeR_".$value;
		}
		return array_unique($all);
	}
	
	private function _comment(): array
	{
		$all = [];
		foreach(Itf()->get('_Comment_ShortCodeR') as $value){
			$all[]="ShortCodeR_".$value;
		}
		return array_unique($all);
	}
	
	private function _topic(): array
	{
		$all = [];
		foreach(Itf()->get('_Topic_ShortCodeR') as $value){
			$all[]="ShortCodeR_".$value;
		}
		return array_unique($all);
	}
	
	private function diff(array $data){
		$arr = [];
		foreach($data as $k=>$v) {
			if(!in_array($k,$this->_())){
				$arr[$k] = $v;
			}
		}
		return $arr;
	}
	
	private function diff_comment(array $data)
	{
		$arr = [];
		foreach($data as $k=>$v) {
			if(!in_array($k,$this->_comment())){
				$arr[$k] = $v;
			}
		}
		return $arr;
	}
	
	private function diff_topic(array $data)
	{
		$arr = [];
		foreach($data as $k=>$v) {
			if(!in_array($k,$this->_topic())){
				$arr[$k] = $v;
			}
		}
		return $arr;
	}
}