<?php

declare (strict_types=1);
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $topic_id 
 * @property string $user_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicUpdated extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic_updated';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','user_id','topic_id','created_at','updated_at','user_agent','user_ip'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function user(){
        return $this->belongsTo(User::class,"user_id","id");
    }
}