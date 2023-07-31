<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Crontab;

use App\Model\AdminOption;
use App\Plugins\User\src\Models\UsersPm;
use Hyperf\Crontab\Annotation\Crontab;
/**
 * 用户私信清理.
 */
#[Crontab(name: 'CleanUsersPm', rule: '0 */12 * * *', callback: 'execute', memo: '用户私信清理', enable: ['App\\Plugins\\Core\\src\\Crontab\\CleanUsersPm', 'isEnable'])]
class CleanUsersPm
{
    public function execute() : void
    {
        // 消息保留时间,单位:秒
        $reserve = (int) get_options('pm_msg_reserve', 7) * 24 * 60 * 60;
        foreach (UsersPm::query()->get() as $pm) {
            if (time() - strtotime($pm->created_at) >= $reserve) {
                UsersPm::query()->where('id', $pm->id)->delete();
            }
        }
    }
    public function isEnable() : bool
    {
        return !((int) $this->get_options('pm_msg_reserve', 7) === 0);
    }
    private function core_default($string = null, $default = null)
    {
        if ($string) {
            return $string;
        }
        return $default;
    }
    private function get_options($name, $default = '')
    {
        if (!cache()->has('admin.options.' . $name)) {
            cache()->set('admin.options.' . $name, @AdminOption::query()->where('name', $name)->first()->value);
        }
        return $this->core_default(cache()->get('admin.options.' . $name), $default);
    }
}