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
    protected $table = 'users_options';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id', 'created_at', 'updated_at', 'qianming', 'qq', 'weixin', 'website', 'email', 'options', 'credits', 'golds', 'exp', 'money'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    /**
     * 设置用户的余额.
     *
     * @param float|int|string $value
     */
    public function setMoneyAttribute($value)
    {
        $this->attributes['money'] = ((float) $value) * 100;
    }

    /**
     * 获取用户的余额.
     *
     * @param float|int|string $value
     */
    public function getMoneyAttribute($value)
    {
        return ((float) $value) / 100;
    }

    /**
     * 获取用户的积分.
     * @param $value
     * @return int
     */
    public function getCreditsAttribute($value)
    {
        return ((float) $value)/100;
    }

    /**
     * 设置用户的积分.
     * @param $value
     * @return void
     */
    public function setCreditsAttribute($value)
    {
        $this->attributes['credits'] = ((float) $value)*100;
    }

    /**
     * 获取用户的金币
     * @param $value
     * @return float
     */
    public function getGoldsAttribute($value)
    {
        return ((float) $value)/100;
    }

    /**
     * 设置用户的金币
     * @param $value
     * @return void
     */
    public function setGoldsAttribute($value)
    {
        $this->attributes['golds'] = ((float) $value)*100;
    }

    // 获取用户经验
    public function getExpAttribute($value)
    {
        return ((int) $value);
    }

    // 设置用户经验
    public function setExpAttribute($value)
    {
        $this->attributes['exp'] = ((int) $value);
    }
}
