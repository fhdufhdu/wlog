use uuid::Uuid;

#[derive(sqlx::FromRow)]
pub struct OrphanImage {
    pub id: Uuid,
    pub storage_name: String,
}
