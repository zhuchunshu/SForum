<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\CodeFec;

use Hyperf\Utils\Arr;
use Noodlehaus\Config;

class Themes
{
    public function get(): array
    {
        $arr = getPath(theme_path());
        $plugin_arr = [];
        foreach ($arr as $value) {
            if (
                file_exists(theme_path($value . '/' . $value . '.json'))
                && is_dir(theme_path($value . '/resources'))
                && is_dir(theme_path($value . '/resources/views'))
            ) {
                // 主题目录名
                $plugin_arr[$value]['dir'] = $value;
                // 主题路径
                $plugin_arr[$value]['path'] = theme_path($value);
                // 主题信息
                $plugin_arr[$value]['data'] = Config::load(theme_path($value . '/' . $value . '.json'))->all();

                $plugin_arr[$value]['file'] = theme_path($value . '/' . $value . '.php');
            }
        }
        return $plugin_arr;
    }

    /**
     * @param string $dirName 插件目录名
     * @return ?string
     */
    public function getLogo(string $dirName): ?string
    {
        // 插件logo
        $file = theme_path($dirName . '/' . $dirName . '.png');
        if (file_exists($file)) {
            $image_info = getimagesize($file);
            $image_data = fread(fopen($file, 'rb'), filesize($file));
            return 'data:' . $image_info['mime'] . ';base64,' . chunk_split(base64_encode($image_data));
        }
        return null;
    }

    /**
     * 判断主题是否存在.
     * @param string $name 主题名
     */
    public function has(string $name): bool
    {
        return Arr::has($this->get(), $name);
    }
}
