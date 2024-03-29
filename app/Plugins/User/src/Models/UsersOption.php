<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id
 * @property string $qianming
 * @property string $qq
 * @property string $wx
 * @property string $website
 * @property string $email
 * @property string $options
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UsersOption extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'users_options';

    /**
     * The attributes that are mass assignable.
     */
    protected array $fillable = ['id', 'created_at', 'updated_at', 'qianming', 'qq', 'weixin', 'website', 'email', 'options', 'credits', 'golds', 'exp', 'money'];

    /**
     * The attributes that should be cast to native types.
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    /**
     * 设置用户的余额.
     *
     * @param float|int|string $value
     */
    public function setMoneyAttribute($value)
    {
        $this->attributes['money'] = (float) $value * 100;
    }

    /**
     * 获取用户的余额.
     *
     * @param float|int|string $value
     */
    public function getMoneyAttribute($value)
    {
        return (float) $value / 100;
    }

    /**
     * 获取用户的积分.
     * @param $value
     * @return int
     */
    public function getCreditsAttribute($value)
    {
        return (float) $value / 100;
    }

    /**
     * 设置用户的积分.
     * @param $value
     */
    public function setCreditsAttribute($value)
    {
        $this->attributes['credits'] = (float) $value * 100;
    }

    /**
     * 获取用户的金币
     * @param $value
     * @return float
     */
    public function getGoldsAttribute($value)
    {
        return (float) $value / 100;
    }

    /**
     * 设置用户的金币
     * @param $value
     */
    public function setGoldsAttribute($value)
    {
        $this->attributes['golds'] = (float) $value * 100;
    }

    // 获取用户经验
    public function getExpAttribute($value)
    {
        return (float) $value;
    }

    // 设置用户经验
    public function setExpAttribute($value)
    {
        $this->attributes['exp'] = (float) $value;
    }

    // 增加用户积分
    public function addCredits($value)
    {
        $this->credits += (float) $value;
        $this->save();
    }

    // 扣除用户积分
    public function reduceCredits($value)
    {
        $this->credits -= (float) $value;
        $this->save();
    }

    // 增加用户金币
    public function addGolds($value)
    {
        $this->golds += (float) $value;
        $this->save();
    }

    // 扣除用户金币
    public function reduceGolds($value)
    {
        $this->golds -= (float) $value;
        $this->save();
    }

    // 获取用户
    public function user()
    {
        return $this->hasOne(User::class, 'options_id');
    }

    // 处理获取签名
    public function getQianmingAttribute($value): string
    {
        // 用户组id
        $class_id = $this->user->class_id;
        if ((int) get_options('user_black_group_id') === (int) $class_id) {
            return get_options('user_ban_re_qianming', '该用户已被封禁');
        }
        return $value;
    }
}
