<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id
 * @property string $content
 * @property string $user_agent
 * @property string $user_ip
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class Post extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'posts';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id', 'user_id', 'comment_id', 'topic_id', 'content', 'user_agent', 'user_ip', 'created_at', 'updated_at'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    /**
     * 获取作者信息.
     */
    public function user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }

    /**
     * 获取所属帖子信息.
     */
    public function topic(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(Topic::class, 'topic_id', 'id');
    }

    /**
     * 获取所属评论信息.
     */
    public function comments(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(TopicComment::class, 'comment_id', 'id');
    }

    /*
     * 获取options信息
     */
    public function options(): \Hyperf\Database\Model\Relations\HasOne
    {
        return $this->hasOne(PostsOption::class, 'post_id');
    }
}
