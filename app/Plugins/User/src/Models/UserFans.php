<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;
/**
 * @property int $id 
 * @property string $user_id 
 * @property string $fans_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UserFans extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'users_fans';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['user_id', 'fans_id', 'created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
    public function fans()
    {
        return $this->belongsTo(User::class, "fans_id", "id");
    }
    public function user()
    {
        return $this->belongsTo(User::class, "user_id", "id");
    }
}