<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib;

use Hyperf\Utils\Arr;
use Noodlehaus\Config;

class Emoji
{
    /**
     * 获取所有emoji.
     */
    public function get(): array
    {
        $all = Itf()->get('emoji') ?: [];
        $emoji = [];
        foreach ($all as $data) {
            if (Arr::has($data, 'name') && Arr::has($data, 'emoji') && file_exists($data['emoji'])) {
                $emoji[] = $data;
            }
        }
        $emojis = [];
        foreach ($emoji as $data) {
            $allEmoji = Config::load($data['emoji'])->all() ?: [];
            $allEmojis = [];
            switch ($data['type']) {
                case 'emoji':
                    foreach ($allEmoji as $k => $v) {
                        $allEmojis[] = [
                            'icon' => $k,
                            'text' => $k,
                        ];
                    }
                    break;
                case 'image':
                    foreach ($allEmoji as $k => $v) {
                        $allEmojis[] = [
                            'icon' => '<img class="emoji-tippy" alt="' . $k . '" min-width="' . get_options('contentParse_owo_width', 50) . '" min-height="' . get_options('contentParse_owo_height', 50) . '" src="' . $v . '" />',
                            'text' => '::' . $data['name'] . ':' . $k . '::',
                        ];
                    }
                    break;
                default:
                    foreach ($allEmoji as $k => $v) {
                        $allEmojis[] = [
                            'icon' => $v,
                            'text' => $v,
                        ];
                    }
                    break;
            }
            if (is_array($allEmojis) && count($allEmojis)) {
                $emojis[$data['name']] = [
                    'type' => $data['type'],
                    'container' => $allEmojis,
                ];
            }
        }
        return $emojis;
    }

    /**
     * 获取所有图片表情.
     */
    public function getImg(): array
    {
        $all = Itf()->get('emoji') ?: [];
        $emoji = [];
        foreach ($all as $data) {
            if (Arr::has($data, 'name') && Arr::has($data, 'emoji') && file_exists($data['emoji'])) {
                $emoji[] = $data;
            }
        }
        $emojis = [];
        foreach ($emoji as $data) {
            if (! Arr::has($data, 'type') || @$data['type'] !== 'image') {
                $allEmoji = Config::load($data['emoji'])->all() ?: [];
                if (is_array($allEmoji) && count($allEmoji)) {
                    $emojis['text'][$data['name']] = $allEmoji;
                }
            } else {
                $allEmoji = Config::load($data['emoji'])->all() ?: [];
                if (is_array($allEmoji) && count($allEmoji)) {
                    $emojis[$data['type']][$data['name']] = $allEmoji;
                }
            }
        }
        return $emojis;
    }

    public function parse(string $content): array | string | null
    {
        return (new \Zhuchunshu\EmojiParse\Emoji())->parse($content, function ($match) {
            $name = $match[1];
            $emoji = $match[2];
            if ($this->liftSize($name)) {
                if (get_options('contentParse_owo_img', '开启') === '开启') {
                    return '<img alt="' . $match[0] . '" max-width="100" height="150" src="' . $this->getImg()['image'][$name][$emoji] . '" />';
                }
            } elseif (get_options('contentParse_owo_img', '开启') === '开启') {
                return '<img alt="' . $match[0] . '" width="' . get_options('contentParse_owo_width', 30) . '" height="' . get_options('contentParse_owo_height', 30) . '" src="' . $this->getImg()['image'][$name][$emoji] . '" />';
            }

            return $match[0];
        });
    }

    private function liftSize($name)
    {
        foreach (Itf()->get('emoji') as $item) {
            if ($item['name'] === $name && $item['size'] === true) {
                return true;
            }
        }
        return false;
    }
}
