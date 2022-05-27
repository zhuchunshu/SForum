<?php

namespace App\Plugins\Core\src\Lib;

use Hyperf\Utils\Arr;
use Noodlehaus\Config;

class Emoji
{
	/**
	 * 获取所有emoji
	 * @return array
	 */
	public function get(): array
	{
		$all = Itf()->get('emoji') ?: [];
		$emoji = [];
		foreach($all as $data) {
			if(Arr::has($data, 'name') && Arr::has($data, 'emoji') && file_exists($data['emoji'])) {
				$emoji[] = $data;
			}
		}
		$emojis = [];
		foreach($emoji as $data) {
			if(!Arr::has($data, 'type') || @$data['type'] !== 'img') {
				$allEmoji = Config::load($data['emoji'])->all() ?: [];
				if(is_array($allEmoji) && count($allEmoji)) {
					$emojis['text'][$data['name']] = $allEmoji;
				}
			} else {
				$allEmoji = Config::load($data['emoji'])->all() ?: [];
				if(is_array($allEmoji) && count($allEmoji)) {
					$emojis[$data['type']][$data['name']] = $allEmoji;
				}
			}
		}
		return $emojis;
		
	}
	
	
	public function parse(string $content): array|string|null
	{
		return (new \Zhuchunshu\EmojiParse\Emoji())->parse($content, function($match) {
			$name = $match[1];
			$emoji = $match[2];
			if(get_options('contentParse_owo_text', '开启') === "开启" && Arr::has($this->get(), 'text') && Arr::has($this->get()['text'], $name) && Arr::has($this->get()['text'][$name], $emoji)) {
				return $this->get()['text'][$name][$emoji];
			}
			if(get_options('contentParse_owo_img', '开启') === "开启" && Arr::has($this->get(), 'img') && Arr::has($this->get()['img'], $name) && Arr::has($this->get()['img'][$name], $emoji)) {
				return '<img alt="' . $match[0] . '" width="' . get_options("contentParse_owo_width", 25) . '" height="' . get_options("contentParse_owo_height", 25) . '" src="' . $this->get()['img'][$name][$emoji] . '" />';
			}
			return $match[0];
		});
		
	}
}