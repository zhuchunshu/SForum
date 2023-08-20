<?php

namespace App\CodeFec;

use Hyperf\Collection\Arr;
use Noodlehaus\Config;

class Language
{
	/**
	 * 获取所有语言
	 * @return array
	 */
	public function all(): array
	{
		$arr = getPath(lang_path());
		$langs = [];
		foreach($arr as $value) {
			if(
				file_exists(lang_path($value . "/data.json"))
			) {
				$langs[$value] = Config::load(lang_path($value . "/data.json"))->get('name', $value);
			}
		}
		return $langs;
	}
	
	/**
	 * 判断语言是否存在
	 * @param string $language
	 * @return bool
	 */
	public function has(string $language): bool
	{
		return Arr::has($this->all(), $language);
	}
}