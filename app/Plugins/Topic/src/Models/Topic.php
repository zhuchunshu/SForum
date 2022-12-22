<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Core\src\Models\PostsOption;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id
 * @property string $title
 * @property string $user_id
 * @property string $status
 * @property string $content
 * @property string $markdown
 * @property string $like
 * @property string $view
 * @property string $tag_id
 * @property string $_token
 * @property string $options
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class Topic extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id', 'created_at', 'updated_at', 'title', 'user_id', 'status', 'post_id', 'view', 'tag_id', 'options', '_token', 'topping', 'essence'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    /**
     * 帖子标签信息.
     */
    public function tag(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(TopicTag::class, 'tag_id', 'id');
    }

    /**
     * 帖子作者信息.
     */
    public function user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }

    /**
     * 帖子更新记录.
     */
    public function topic_updated(): \Hyperf\Database\Model\Relations\HasMany
    {
        return $this->hasMany(TopicUpdated::class, 'topic_id');
    }

    /**
     * 帖子下的所有评论.
     */
    public function comments(): \Hyperf\Database\Model\Relations\HasMany
    {
        return $this->hasMany(TopicComment::class, 'topic_id');
    }

    /**
     * 帖子下的所有点赞.
     */
    public function likes(): \Hyperf\Database\Model\Relations\HasMany
    {
        return $this->hasMany(TopicLike::class, 'topic_id');
    }

    /**
     * 帖子内容.
     * @return \Hyperf\Database\Model\Relations\BelongsTo
     */
    public function post()
    {
        return $this->belongsTo(Post::class, 'post_id', 'id');
    }
}
