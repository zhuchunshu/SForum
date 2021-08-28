<?php


namespace App\Plugins\Core\src\Lib\ShortCode;


class Make
{

    public function default(string $content):string{
        foreach(ShortCode()->all() as $tag=>$value){
            $tag = core_Itf_id("ShortCode",$tag);

            $pattern = "/\[$tag\](.*?)\[\/$tag\]/is";

            $content = preg_replace_callback($pattern, function($match)use($value){
                return ShortCode()->callback($value['callback'],$match);
            },$content);
        }
        return $content;
    }

    public function type2(string $content):string{
        foreach(ShortCode()->all() as $tag=>$value){
            $tag = core_Itf_id("ShortCode",$tag);

            $pattern = "/\[$tag=(.*?)\](.*?)\[\/$tag\]/is";

            $content = preg_replace_callback($pattern, function($match)use($value){
                return ShortCode()->callback($value['callback'],$match);
            },$content);
        }
        return $content;
    }

    public function type1(string $content):string{
        foreach(ShortCode()->all() as $tag=>$value){
            $tag = core_Itf_id("ShortCode",$tag);

            $pattern = "/\[$tag (.*?)\](.*?)\[\/$tag\]/is";

            $content = preg_replace_callback($pattern, function($match)use($value){
                return ShortCode()->callback($value['callback'],$match);
            },$content);
        }
        return $content;
    }


}