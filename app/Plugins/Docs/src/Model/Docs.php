<?php

declare (strict_types=1);
namespace App\Plugins\Docs\src\Model;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $class_id 
 * @property string $user_id 
 * @property string $title 
 * @property string $quanxian 
 * @property string $content 
 * @property string $markdown 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class Docs extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'docs';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','class_id','user_id','title','content','markdown','created_at', 'updated_at'];
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

    public function docsClass(): \Hyperf\Database\Model\Relations\BelongsTo
    {
        return $this->belongsTo(DocsClass::class,'class_id','id');
    }
}