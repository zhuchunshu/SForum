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
use Hyperf\Database\Model\SoftDeletes;

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
    use SoftDeletes;

    /**
     * The table associated with the model.
     *
     * @var ?string
     */
    protected ?string $table = 'posts';

    /**
     * The attributes that are mass assignable.
     */
    protected array $fillable = ['id', 'user_id', 'comment_id', 'topic_id', 'content', 'user_agent', 'user_ip', 'created_at', 'updated_at'];

    /**
     * The attributes that should be cast to native types.
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

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

    // 处理帖子内容
    public function getContentAttribute($value): string
    {
        @$class_id = $this->user->class_id;
        if ((int) get_options('user_black_group_id') === (int) $class_id) {
            return get_options('user_ban_re_post_content', <<<'HTML'
<div class="alert alert-danger m-0">
  该用户和其发布的内容已被放进小黑屋
</div>
HTML);
        }
        return $value;
    }
}
