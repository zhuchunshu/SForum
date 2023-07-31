<?php

namespace App\Plugins\Core\src\Lib\ShortCodeR;

class Make
{
    public function default(array $all, string $content) : string
    {
        foreach ($all as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $pattern = "/\\[{$tag}\\](.*?)\\[\\/{$tag}\\]/is";
            $content = preg_replace_callback($pattern, function ($match) use($value) {
                return ShortCodeR()->callback($value['callback'], $match);
            }, $content);
        }
        return $content;
    }
    public function type2(array $all, string $content) : string
    {
        foreach ($all as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $pattern = "/\\[{$tag}=(.*?)\\](.*?)\\[\\/{$tag}\\]/is";
            $content = preg_replace_callback($pattern, function ($match) use($value) {
                return ShortCodeR()->callback($value['callback'], $match);
            }, $content);
        }
        return $content;
    }
    public function type1(array $all, string $content) : string
    {
        foreach ($all as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $pattern = "/\\[{$tag} (.*?)\\](.*?)\\[\\/{$tag}\\]/is";
            $content = preg_replace_callback($pattern, function ($match) use($value) {
                return ShortCodeR()->callback($value['callback'], $match);
            }, $content);
        }
        return $content;
    }
    public function type3(array $all, string $content) : string
    {
        foreach ($all as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $pattern = "/\\[{$tag}\\]/is";
            $content = preg_replace_callback($pattern, function ($match) use($value) {
                return ShortCodeR()->callback($value['callback'], $match);
            }, $content);
        }
        return $content;
    }
    public function type4(array $all, string $content) : string
    {
        foreach ($all as $tag => $value) {
            $tag = core_Itf_id("ShortCodeR", $tag);
            $pattern = "/\\[{$tag}=(.*?)\\]/is";
            $content = preg_replace_callback($pattern, function ($match) use($value) {
                return ShortCodeR()->callback($value['callback'], $match);
            }, $content);
        }
        return $content;
    }
}