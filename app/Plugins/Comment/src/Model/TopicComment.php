<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Model;

use App\Model\Model;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;
use Hyperf\Database\Model\SoftDeletes;

/**
 * @property int $id
 * @property int $likes
 * @property string $topic_id
 * @property string $user_id
 * @property string $parent_id
 * @property string $content
 * @property string $status
 * @property string $shenping
 * @property string $optimal
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicComment extends Model
{
    use SoftDeletes;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic_comment';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['topic_id', 'post_id', 'user_id', 'parent_id', 'status', 'shenping', 'optimal', 'parent_url', 'created_at', 'updated_at'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'likes' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime', 'optimal' => 'datetime', 'shenping' => 'datetime'];

    protected $hidden = ['user_ip'];

    /**
     * 评论作者信息.
     */
    public function user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }

    /**
     * 评论所在的帖子信息.
     */
    public function topic(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(Topic::class, 'topic_id', 'id');
    }

    /**
     * 评论parent信息.
     */
    public function parent(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(__CLASS__, 'parent_id', 'id');
    }

    /**
     * 评论内容.
     * @return \Hyperf\Database\Model\Relations\BelongsTo
     */
    public function post()
    {
        return $this->belongsTo(Post::class, 'post_id', 'id');
    }

    /**
     * 评论点赞信息.
     * @return
     */
    public function likes()
    {
        return $this->hasMany(TopicCommentLike::class, 'comment_id', 'id');
    }
}
