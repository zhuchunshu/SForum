<?php

namespace App\Plugins\Core\src\Lib\ShortCodeR;

class Filter
{
	public function default(string $content):string{
		foreach(ShortCodeR()->all() as $tag=>$value){
			$tag = core_Itf_id("ShortCodeR",$tag);
			
			$pattern = "/\[$tag\](.*?)\[\/$tag\]/is";
			
			$content = preg_replace_callback($pattern, function($match)use($tag){
				return $this->filter($match,$tag);
			},$content);
		}
		return $content;
	}
	
	public function type2(string $content):string{
		foreach(ShortCodeR()->all() as $tag=>$value){
			$tag = core_Itf_id("ShortCodeR",$tag);
			
			$pattern = "/\[$tag=(.*?)\](.*?)\[\/$tag\]/is";
			
			$content = preg_replace_callback($pattern, function($match)use($tag){
				return $this->filter($match,$tag);
			},$content);
		}
		return $content;
	}
	
	public function type1(string $content):string{
		foreach(ShortCodeR()->all() as $tag=>$value){
			$tag = core_Itf_id("ShortCodeR",$tag);
			
			$pattern = "/\[$tag (.*?)\](.*?)\[\/$tag\]/is";
			
			$content = preg_replace_callback($pattern, function($match)use($tag){
				return $this->filter($match,$tag);
			},$content);
		}
		return $content;
	}
	
	public function type3(string $content):string{
		foreach(ShortCodeR()->all() as $tag=>$value){
			$tag = core_Itf_id("ShortCodeR",$tag);
			
			$pattern = "/\[$tag\]/is";
			
			$content = preg_replace_callback($pattern, function($match)use($tag){
				return $this->filter($match,$tag);
			},$content);
		}
		return $content;
	}
	
	public function type4(string $content):string{
		foreach(ShortCodeR()->all() as $tag=>$value){
			$tag = core_Itf_id("ShortCodeR",$tag);
			
			$pattern = "/\[$tag=(.*?)\]/is";
			
			$content = preg_replace_callback($pattern, function($match)use($tag){
				return $this->filter($match,$tag);
			},$content);
		}
		return $content;
	}
	
	
	public function filter($match,$tag){
		return '['.$tag."]"."安全性问题,禁止预览此标签内容[/".$tag."]";
	}
}