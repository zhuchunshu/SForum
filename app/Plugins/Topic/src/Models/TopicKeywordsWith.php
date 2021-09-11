<?php

declare (strict_types=1);
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $topic_id 
 * @property string $with_id 
 * @property string $user_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicKeywordsWith extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic_keywords_with';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ["id","topic_id","user_id","with_id","created_at","updated_at"];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function topic(){
        return $this->belongsTo(Topic::class,"topic_id");
    }
}