<?php

declare (strict_types=1);
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $name 
 * @property string $color 
 * @property string $icon 
 * @property string $description 
 * @property string $type 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicTag extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'topic_tag';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ["name","icon","description","color","type",'userClass',"created_at","updated_at","user_id"];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	public function user(): \Hyperf\Database\Model\Relations\BelongsTo
	{
		return $this->belongsTo(User::class,'user_id','id');
	}
}