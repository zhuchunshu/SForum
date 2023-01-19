<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src;

use Hyperf\Utils\Arr;
use Zhuchunshu\EmojiParse\Emoji;

class ContentParse
{
    /**
     * @param string $content 内容
     * @param array $data 数据
     */
    public function parse(string $content, array $data = [])
    {
        $content = $this->ShortCode($content, $data);
        $content = $this->twemoji($content);
        return $this->owo($content);
    }

    /**
     * ShortCode 处理.
     * @param string $content 内容
     * @param array $data 数据
     */
    private function ShortCode(string $content, array $data = []): string
    {
        // 要删除的ShortCode
        // 要删除的ShortCode
        $shortCode = [];
        if (count(Itf()->get('topic_shortCode_remove'))) {
            $shortCode = array_merge($shortCode, Itf()->get('topic_shortCode_remove'));
        }
        if (Arr::has($data, 'remove_shortCode') && count($data['remove_shortCode'])) {
            $shortCode = array_merge($shortCode, $data['remove_shortCode']);
        }
        $shortCode = array_unique($shortCode);
        return ShortCodeR()->setRemove($shortCode)->handle($content, $data);
    }

    /**
     * TwEmoji 处理.
     */
    private function twemoji(string $content): string
    {
        if (get_options('contentParse_twemoji', '开启') === '开启') {
            return (new Emoji())->twemoji($content)->svg()->base(get_options('contentParse_twemoji_cdn', 'https://lib.baomitu.com/twemoji/1.4.2'))->toHtml(null, ['width' => get_options('contentParse_twemoji_contentParse_width', 25), 'height' => get_options('contentParse_twemoji_contentParse_height', 25)]);
        }
        return $content;
    }

    /**
     * 渲染owo表情.
     */
    private function owo(string $content): string
    {
        if (get_options('contentParse_owo', '开启') === '开启') {
            return (new \App\Plugins\Core\src\Lib\Emoji())->parse($content);
        }
        return $content;
    }
}
