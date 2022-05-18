<?php


namespace App\Plugins\Core\src\Lib\ShortCode;


use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\Utils\Arr;
use Hyperf\Utils\Str;
use JetBrains\PhpStorm\Pure;
use Thunder\Shortcode\HandlerContainer\HandlerContainer;
use Thunder\Shortcode\Parser\RegularParser;
use Thunder\Shortcode\Processor\Processor;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;

class ShortCode
{
	
	public HandlerContainer $handlers;
	
	#[Pure] public function __construct(){
		$this->handlers = new HandlerContainer();
		
	}
    public function add($tag,$callback): void
    {
        Itf()->add("ShortCode",$tag,["callback" => $callback]);
    }

    public function all(): array
    {
	    $arr = Itf()->get("ShortCode");
	    $shortCodeR = AnnotationCollector::getMethodsByAnnotation(\App\CodeFec\Annotation\ShortCode\ShortCode::class);
	    foreach ($shortCodeR as $data){
		    $name = $data['annotation']->name;
		    $callback = $data['class']."@".$data['method'];
		    $arr["ShortCode_".$name]=['callback' => $callback];
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
        if(Arr::has($this->all(),"ShortCode_".$tag)){
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
	    foreach($this->all() as $tag=>$value){
		    $tag = core_Itf_id("ShortCode",$tag);
		    $this->handlers->add($tag, function(ShortcodeInterface $s)use($value){
			    $match = [$s->getContent(),$s->getContent(),$s->getContent()];
			    return $this->callback($value['callback'],$match,$s);
		    });
	    }
	    $processor = new Processor(new RegularParser(), $this->handlers);
	    return $processor->process($content);
    }
	

    public function callback($callback,...$parameter){
        $class = Str::before($callback,"@");
        $method = Str::after($callback,"@");
        return (new $class())->$method(...$parameter);
    }
}