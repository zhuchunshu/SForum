<?php

declare (strict_types=1);
namespace App\Plugins\Comment\src\Model;

use App\Model\Model;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property int $likes 
 * @property string $topic_id 
 * @property string $user_id 
 * @property string $parent_id 
 * @property string $content 
 * @property string $markdown 
 * @property string $status 
 * @property string $shenping 
 * @property string $optimal 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicComment extends Model
{
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
    protected $fillable = ['likes','topic_id','user_id','parent_id','content','markdown','status','shenping','optimal','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'likes' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime','optimal' => 'datetime','shenping' => 'datetime'];

    public function user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class,"user_id","id");
    }

    public function topic(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(Topic::class,"topic_id","id");
    }
}