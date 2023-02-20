<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;

/**
 * @property int $id
 * @property int $tag_id
 * @property int $user_id
 */
class Moderator extends Model
{
    public $timestamps = false;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'moderator';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id', 'tag_id', 'user_id'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'tag_id' => 'integer', 'user_id' => 'integer'];

    /**
     * 用户信息.
     */
    public function user(): \Hyperf\Database\Model\Relations\HasOne
    {
        return $this->hasOne(\App\Plugins\User\src\Models\User::class, 'id', 'user_id');
    }

    /**
     * 标签信息.
     */
    public function tag(): \Hyperf\Database\Model\Relations\HasOne
    {
        return $this->hasOne(\App\Plugins\Topic\src\Models\TopicTag::class, 'id', 'tag_id');
    }
}
