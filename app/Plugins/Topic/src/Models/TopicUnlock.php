<?php

declare(strict_types=1);

namespace App\Plugins\Topic\src\Models;



use App\Model\Model;

/**
 * @property int $id 
 * @property int $topic_id 
 * @property \Carbon\Carbon $created_at 
 * @property \Carbon\Carbon $updated_at 
 */
class TopicUnlock extends Model
{
    /**
     * The table associated with the model.
     */
    protected ?string $table = 'topic_unlock';

    /**
     * The attributes that are mass assignable.
     */
    protected array $fillable = ['id','topic_id','created_at','updated_at'];

    /**
     * The attributes that should be cast to native types.
     */
    protected array $casts = ['id' => 'integer', 'topic_id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function topic(){
        return $this->belongsTo(Topic::class,'topic_id','id');
    }
}
