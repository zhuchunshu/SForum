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
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Moderator;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicTag;
use Carbon\Carbon;
use Qbhy\HyperfAuth\AuthAbility;

/**
 * @property int $id
 * @property string $username
 * @property string $email
 * @property string $password
 * @property string $avatar
 * @property string $email_ver_time
 * @property string $phone_ver_time
 * @property string $class_id
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class User extends Model implements \Qbhy\HyperfAuth\Authenticatable
{
    use AuthAbility;

    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'users';

    /**
     * The attributes that are mass assignable.
     */
    protected array $fillable = ['username', 'password', 'email', 'avatar', 'class_id', 'email_ver_time', 'phone_ver_time', '_token', 'options_id'];

    /**
     * The attributes that should be hidden for arrays.
     */
    protected array $hidden = ['password', '_token', 'email', 'phone', 'phone_ver_time', 'email_ver_time'];

    /**
     * The attributes that should be cast to native types.
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function Class()
    {
        return $this->belongsTo(UserClass::class, 'class_id', 'id');
    }

    public function options()
    {
        return $this->belongsTo(UsersOption::class, 'options_id', 'id');
    }

    public function users_option()
    {
        return $this->belongsTo(UsersOption::class, 'options_id', 'id');
    }

    /**
     * 获取用户的评论.
     * @return \Hyperf\Database\Model\Relations\HasMany
     */
    public function comments()
    {
        return $this->hasMany(TopicComment::class, 'user_id', 'id');
    }

    /**
     * 获取用户的话题.
     * @return \Hyperf\Database\Model\Relations\HasMany
     */
    public function topic()
    {
        return $this->hasMany(Topic::class, 'user_id', 'id');
    }

    /**
     * 收藏.
     * @return \Hyperf\Database\Model\Relations\HasMany
     */
    public function collections()
    {
        return $this->hasMany(UsersCollection::class, 'user_id', 'id');
    }

    /**
     * 粉丝.
     * @return \Hyperf\Database\Model\Relations\HasMany
     */
    public function fan()
    {
        return $this->hasMany(UserFans::class, 'user_id', 'id');
    }

    /**
     * 主题板块.
     * @return \Hyperf\Database\Model\Relations\HasMany
     */
    public function tags()
    {
        return $this->hasMany(TopicTag::class, 'user_id', 'id');
    }

    public function auth()
    {
        return $this->hasMany(UsersAuth::class, 'user_id', 'id');
    }

    public function moderator()
    {
        return $this->hasMany(Moderator::class, 'user_id', 'id');
    }

    public function scopeRegisteredBefore($query, $timestamp)
    {
        return $query->where('created_at', '<=', date('Y-m-d H:i:s', $timestamp));
    }

    // 处理获取用户名
    public function getUsernameAttribute($value): string
    {
        // 用户组id
        $class_id = $this->attributes['class_id'];
        if ((int) get_options('user_black_group_id') === (int) $class_id) {
            return 'ban#' . numberToUniqueLetter($this->attributes['id']);
        }
        return $value;
    }

    // 处理获取头像
    public function getAvatarAttribute($value): ?string
    {
        // 用户组id
        $class_id = $this->attributes['class_id'];
        if ((int) get_options('user_black_group_id') === (int) $class_id) {
            return get_options('user_ban_re_avatar', '/plugins/Core/image/ban_user.png');
        }
        return $value;
    }
}
