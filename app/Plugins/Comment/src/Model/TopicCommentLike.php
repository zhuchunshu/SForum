<?php

declare (strict_types=1);
namespace App\Plugins\Comment\src\Model;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $comment_id 
 * @property string $user_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicCommentLike extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic_comment_likes';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','user_id','comment_id','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}