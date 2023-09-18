<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Plugins\Core\src\Lib\Authority\Authority;
use App\Plugins\Core\src\Lib\Redirect;
use App\Plugins\Core\src\Lib\ShortCodeR\ShortCodeR;
use App\Plugins\Core\src\Lib\UserVerEmail;
use App\Plugins\Core\src\Models\PayAmountRecord;
use DivineOmega\PHPSummary\SummaryTool;
use JetBrains\PhpStorm\Pure;

if (! function_exists('plugins_core_user_reg_defuc')) {
    function plugins_core_user_reg_defuc()
    {
        return \App\Plugins\User\src\Models\UserClass::query()->select('id', 'name')->get();
    }
}

if (! function_exists('super_avatar')) {
    function super_avatar($user_data): string
    {
        if ($user_data->avatar) {
            return $user_data->avatar;
        }

        if (get_options('core_user_def_avatar', 'gavatar') !== 'ui-avatars') {
            return get_options('theme_common_gavatar', 'https://cn.gravatar.com/avatar/') . md5($user_data->email);
        }
        return 'https://ui-avatars.com/api/?background=random&format=svg&name=' . $user_data->username;
    }
}

if (! function_exists('avatar')) {
    function avatar($user_data): string
    {
        return super_avatar($user_data);
    }
}

if (! function_exists('redirect')) {
    #[Pure] function redirect(): Redirect
    {
        return new Redirect();
    }
}

if (! function_exists('core_user_ver_email_make')) {
    function core_user_ver_email(): UserVerEmail
    {
        return new UserVerEmail();
    }
}

if (! function_exists('Core_Ui')) {
    function Core_Ui(): App\Plugins\Core\src\Lib\Ui
    {
        return new App\Plugins\Core\src\Lib\Ui();
    }
}

if (! function_exists('core_Str_menu_url')) {
    function core_Str_menu_url(string $path): string
    {
        if ($path === '//') {
            $path = '/';
        }
        return $path;
    }
}

if (! function_exists('core_menu_pd')) {
    function core_menu_pd(string $id)
    {
        foreach (_menu() as $value) {
            if (arr_has($value, 'parent_id') && (string) $value['parent_id'] === (string) $id) {
                return true;
            }
        }
        return false;
    }
}

if (! function_exists('core_Itf_id')) {
    function core_Itf_id($name, $id)
    {
        return \Hyperf\Stringable\Str::after($id, $name . '_');
    }
}

if (! function_exists('core_menu_pdArr')) {
    function core_menu_pdArr($id): array
    {
        $arr = [];
        foreach (_menu() as $key => $value) {
            if (arr_has($value, 'parent_id') && (string) $value['parent_id'] === (string) $id) {
                $arr[$key] = $value;
            }
        }
        return $arr;
    }
}

if (! function_exists('core_default')) {
    function core_default($string = null, $default = null)
    {
        if ($string) {
            return $string;
        }
        return $default;
    }
}

if (! function_exists('markdown')) {
    function markdown(): Parsedown
    {
        return new Parsedown();
    }
}

if (! function_exists('ShortCodeR')) {
    function ShortCodeR(): ShortCodeR
    {
        return new ShortCodeR();
    }
}

if (! function_exists('xss')) {
    function xss(): App\Plugins\Core\src\Lib\Xss\Xss
    {
        return new App\Plugins\Core\src\Lib\Xss\Xss();
    }
}

if (! function_exists('summary')) {
    function summary($content): string
    {
        return (new SummaryTool($content))->getSummary();
    }
}

if (! function_exists('deOptions')) {
    function deOptions($json)
    {
        return json_decode($json, true);
    }
}

if (! function_exists('getAllImg')) {
    function getAllImg($content): array
    {
        $preg = '/<img.*?src=[\"|\']?(.*?)[\"|\']?\s.*?>/i'; //匹配img标签的正则表达式

        preg_match_all($preg, $content, $allImg); //这里匹配所有的imgecho
        return $allImg[1];
    }
}

if (! function_exists('format_date')) {
    function format_date($time)
    {
        $t = time() - strtotime((string) $time);
        $f = [
            '31536000' => __('app.year'),
            '2592000' => __('app.month'),
            '604800' => __('app.week'),
            '86400' => __('app.day'),
            '3600' => __('app.hour'),
            '60' => __('app.minute'),
            '1' => __('app.second'),
        ];
        foreach ($f as $k => $v) {
            if (0 != $c = floor($t / (int) $k)) {
                return $c . $v . __('app.ago');
            }
        }
    }
}

if (! function_exists('get_all_at')) {
    /**
     * 获取内容中所有被艾特的用户.
     */
    function get_all_at(string $content): array
    {
        preg_match_all('/@(\w+)(?=[^\w@]|$)/u', $content, $arr);
        return $arr[1];
    }
}

if (! function_exists('replace_all_at_space')) {
    function replace_all_at_space(string $content): string
    {
        $pattern = '/@(\w+)(?=\s|$)/u';
        return preg_replace_callback($pattern, static function ($match) {
            return $match[0];
        }, $content);
    }
}

if (! function_exists('remove_all_p_space')) {
    function remove_all_p_space(string $content): string
    {
        return str_replace(' </p>', '</p>', $content);
    }
}

if (! function_exists('replace_all_at')) {
    function replace_all_at(string $content): string
    {
        //$pattern = "/\\$\\[(.*?)]/u";
        $pattern = '/@(\w+)(?=[^\w@]|$)/u';
        $content = replace_all_at_space($content);
        return remove_all_p_space(preg_replace_callback($pattern, static function ($match) {
            var_dump($match);
            return (new \App\Plugins\Core\src\Lib\TextParsing())->at($match[1]);
        }, $content));
    }
}

if (! function_exists('get_all_keywords')) {
    /**
     * 获取内容中所有话题标签.
     */
    function get_all_keywords(string $content): array
    {
        preg_match_all('/#(\p{L}+)/u', $content, $arrMatches);
        return $arrMatches[1];
    }

    function replace_all_keywords(string $content): string
    {
        $pattern = '/#(\p{L}+)/u';
        return preg_replace_callback($pattern, static function ($match) {
            return (new \App\Plugins\Core\src\Lib\TextParsing())->keywords($match[1]);
        }, $content);
    }
}

if (! function_exists('Authority')) {
    function Authority()
    {
        return new Authority();
    }
}

if (! function_exists('curd')) {
    function curd(): App\Plugins\Core\src\Lib\Curd
    {
        return new \App\Plugins\Core\src\Lib\Curd();
    }
}

if (! function_exists('core_http_build_query')) {
    function core_http_build_query(array $data, array $merge): string
    {
        $data = array_merge($data, $merge);
        return http_build_query($data);
    }
}

if (! function_exists('core_http_url')) {
    function core_http_url(): string
    {
        $query = http_build_query(request()->all());
        return request()->path() . '?' . $query;
    }
}

if (! function_exists('core_get_page')) {
    function core_get_page(string $url): array
    {
        $data = explode('=', parse_url($url)['query']);
        return [$data[0] => $data[1]];
    }
}

if (! function_exists('emoji_add')) {
    /**
     * @param string $name emoji name
     * @param string $emoji emoji json path
     * @param string $type emoji type emoji | image | emoticon(颜文字)
     * @throws Exception
     */
    function emoji_add(string $name, string $emoji, string $type, bool $size = false)
    {
        Itf()->add('emoji', count(Itf()->get('emoji')) + 1, [
            'name' => $name,
            'emoji' => $emoji,
            'type' => $type,
            'size' => $size,
        ]);
    }
}

// 截取省份
if (! function_exists('intercept_province')) {
    function intercept_province(string $address)
    {
        $all = \Noodlehaus\Config::load(plugin_path('Core/src/province.json'))->all();
        $province = [];
        foreach ($all as $item) {
            $province[] = $item['name'];
        }
        // 输出
        $echo = null;
        foreach ($province as $item) {
            if (\Hyperf\Stringable\Str::is('*' . $item . '*', $address)) {
                $echo = $item;
            }
        }
        return $echo ?: $address;
    }
}

// 创建余额变更记录
if (! function_exists('create_amount_record')) {
    /**
     * @param  $user_id int|string 用户id
     * @param float|string $origin int|string 变更前
     * @param float|string $cash int|string 变更后
     * @param null|string $type int|string|null 变更类型
     * @param null|float|string $change int|string|null 变更幅度
     * @param null|int|string $order_id string|int 订单号
     * @param null|string $remark string|int 备注
     * @return \Hyperf\Database\Model\Model|PayAmountRecord
     */
    function create_amount_record(int | string $user_id, float | string $origin, float | string $cash, string $type = null, float | string $change = null, int | string $order_id = null, string $remark = null, ): PayAmountRecord | \Hyperf\Database\Model\Model
    {
        return PayAmountRecord::create([
            'user_id' => $user_id,
            'original' => $origin,
            'cash' => $cash,
            'type' => $type,
            'change' => $change,
            'order_id' => $order_id,
            'remark' => $remark,
        ]);
    }
}

if (! function_exists('is_negative')) {
    /**
     * 判断内容为负数.
     * @param float|int|string $number
     * @return bool
     */
    function is_negative(string | float | int $number): bool
    {
        if ($number < 0) {
            return true;
        }
        return str_starts_with($number, '-');
    }
}
