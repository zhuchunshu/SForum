<?php

declare (strict_types=1);
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use App\Plugins\Comment\src\Model\TopicComment;
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
    protected $fillable = ['id','created_at','updated_at','title','user_id','status','content','markdown','view','like','tag_id','options','_token','topping','essence','updated_user','user_agent','user_ip'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function tag(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(TopicTag::class,"tag_id","id");
    }

    public function user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class,"user_id","id");
    }

    public function update_user(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(User::class,"updated_user","id");
    }

    public function topic_updated(): \Hyperf\Database\Model\Relations\HasMany
    {
        return $this->hasMany(TopicUpdated::class,"topic_id");
    }

    public function comments(): \Hyperf\Database\Model\Relations\HasMany
    {
        return $this->hasMany(TopicComment::class,"topic_id");
    }

}